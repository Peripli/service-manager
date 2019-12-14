/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package common

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gavv/httpexpect"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/onsi/ginkgo"
	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

func init() {
	// dummy env to put SM pflags to flags
	TestEnv(SetTestFileLocation)
}

const SMServer = "sm-server"
const OauthServer = "oauth-server"
const BrokerServerPrefix = "broker-"

type TestContextBuilder struct {
	envPreHooks  []func(set *pflag.FlagSet)
	envPostHooks []func(env env.Environment, servers map[string]FakeServer)

	smExtensions       []func(ctx context.Context, smb *sm.ServiceManagerBuilder, env env.Environment) error
	defaultTokenClaims map[string]interface{}
	tenantTokenClaims  map[string]interface{}

	shouldSkipBasicAuthClient bool

	Environment func(f ...func(set *pflag.FlagSet)) env.Environment
	Servers     map[string]FakeServer
	HttpClient  *http.Client
}

type TestContext struct {
	wg            *sync.WaitGroup
	wsConnections []*websocket.Conn

	SM          *SMExpect
	SMWithOAuth *SMExpect
	// Requests a token the the "multitenant" oauth client - then token issued by this client contains
	// the "multitenant" client id behind the specified token claim in the api config
	// the token also contains a "tenant identifier" behind the configured tenant_indentifier claim that
	// will be compared with the value of the label specified in the "label key" configuration
	// In the end requesting brokers with this
	SMWithOAuthForTenant *SMExpect
	SMWithBasic          *SMExpect
	SMRepository         storage.TransactionalRepository

	TestPlatform *types.Platform

	Servers map[string]FakeServer
}

type SMExpect struct {
	*httpexpect.Expect
}

func (expect *SMExpect) List(path string) *httpexpect.Array {
	return expect.ListWithQuery(path, "")
}

func (expect *SMExpect) ListWithQuery(path string, query string) *httpexpect.Array {
	req := expect.GET(path)
	if query != "" {
		req = req.WithQueryString(query)
	}
	page := req.Expect().Status(http.StatusOK).JSON().Object()
	token, hasMoreItems := page.Raw()["token"]
	items := page.Value("items").Array().Raw()

	for hasMoreItems {
		req := expect.GET(path)
		if query != "" {
			req = req.WithQueryString(query)
		}
		page = req.WithQuery("token", token).Expect().Status(http.StatusOK).JSON().Object()
		items = append(items, page.Value("items").Array().Raw()...)
		token, hasMoreItems = page.Raw()["token"]
	}

	return httpexpect.NewArray(ginkgo.GinkgoT(), items)
}

type testSMServer struct {
	cancel context.CancelFunc
	*httptest.Server
}

func (ts *testSMServer) URL() string {
	return ts.Server.URL
}

func (ts *testSMServer) Close() {
	ts.Server.Close()
	ts.cancel()
}

// DefaultTestContext sets up a test context with default values
func DefaultTestContext() *TestContext {
	return NewTestContextBuilder().Build()
}

// NewTestContextBuilder sets up a builder with default values
func NewTestContextBuilder() *TestContextBuilder {
	return &TestContextBuilder{
		envPreHooks: []func(set *pflag.FlagSet){
			SetTestFileLocation,
			SetNotificationsCleanerSettings,
			SetLogOutput,
		},
		Environment: TestEnv,
		envPostHooks: []func(env env.Environment, servers map[string]FakeServer){
			func(env env.Environment, servers map[string]FakeServer) {
				env.Set("api.token_issuer_url", servers["oauth-server"].URL())
			},
			func(env env.Environment, servers map[string]FakeServer) {
				flag.VisitAll(func(flag *flag.Flag) {
					if flag.Value.String() != "" {
						// if any of the go test flags have been set, propagate the value in sm env with highest prio
						// when env exposes the pflagset it would be better to instead override the pflag value instead
						env.Set(flag.Name, flag.Value.String())
					}
				})
			},
		},
		smExtensions:       []func(ctx context.Context, smb *sm.ServiceManagerBuilder, env env.Environment) error{},
		defaultTokenClaims: make(map[string]interface{}, 0),
		tenantTokenClaims:  make(map[string]interface{}, 0),
		Servers: map[string]FakeServer{
			"oauth-server": NewOAuthServer(),
		},
		HttpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   20 * time.Second,
					KeepAlive: 20 * time.Second,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       30 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

func SetTestFileLocation(set *pflag.FlagSet) {
	_, b, _, _ := runtime.Caller(0)
	basePath := path.Dir(b)
	err := set.Set("file.location", basePath)
	if err != nil {
		panic(err)
	}
}

func SetNotificationsCleanerSettings(set *pflag.FlagSet) {
	err := set.Set("storage.notification.clean_interval", "24h")
	if err != nil {
		panic(err)
	}
	err = set.Set("storage.notification.keep_for", "24h")
	if err != nil {
		panic(err)
	}
}

func SetLogOutput(set *pflag.FlagSet) {
	err := set.Set("log.output", "ginkgowriter")
	if err != nil {
		panic(err)
	}
}

func TestEnv(additionalFlagFuncs ...func(set *pflag.FlagSet)) env.Environment {
	// copies all sm pflags to flag so that those can be set via go test
	f := func(set *pflag.FlagSet) {
		if set == nil {
			return
		}

		set.VisitAll(func(pflag *pflag.Flag) {
			if flag.Lookup(pflag.Name) == nil {
				// marker so that if the flag is passed to go test it is recognized
				flag.String(pflag.Name, "", pflag.Usage)
			}
		})
	}

	additionalFlagFuncs = append(additionalFlagFuncs, f)

	env, _ := env.Default(context.TODO(), append([]func(set *pflag.FlagSet){config.AddPFlags}, additionalFlagFuncs...)...)
	return env
}

func (tcb *TestContextBuilder) SkipBasicAuthClientSetup(shouldSkip bool) *TestContextBuilder {
	tcb.shouldSkipBasicAuthClient = shouldSkip

	return tcb
}

func (tcb *TestContextBuilder) WithDefaultEnv(envCreateFunc func(f ...func(set *pflag.FlagSet)) env.Environment) *TestContextBuilder {
	tcb.Environment = envCreateFunc

	return tcb
}

func (tcb *TestContextBuilder) WithAdditionalFakeServers(additionalFakeServers map[string]FakeServer) *TestContextBuilder {
	if tcb.Servers == nil {
		tcb.Servers = make(map[string]FakeServer, 0)
	}

	for name, server := range additionalFakeServers {
		tcb.Servers[name] = server
	}

	return tcb
}

func (tcb *TestContextBuilder) WithTenantTokenClaims(tenantTokenClaims map[string]interface{}) *TestContextBuilder {
	tcb.tenantTokenClaims = tenantTokenClaims

	return tcb
}

func (tcb *TestContextBuilder) WithDefaultTokenClaims(defaultTokenClaims map[string]interface{}) *TestContextBuilder {
	tcb.defaultTokenClaims = defaultTokenClaims

	return tcb
}

func (tcb *TestContextBuilder) WithEnvPreExtensions(fs ...func(set *pflag.FlagSet)) *TestContextBuilder {
	tcb.envPreHooks = append(tcb.envPreHooks, fs...)

	return tcb
}

func (tcb *TestContextBuilder) WithEnvPostExtensions(fs ...func(e env.Environment, servers map[string]FakeServer)) *TestContextBuilder {
	tcb.envPostHooks = append(tcb.envPostHooks, fs...)

	return tcb
}

func (tcb *TestContextBuilder) WithSMExtensions(fs ...func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error) *TestContextBuilder {
	tcb.smExtensions = append(tcb.smExtensions, fs...)

	return tcb
}

func (tcb *TestContextBuilder) Build() *TestContext {
	return tcb.BuildWithListener(nil)
}

func (tcb *TestContextBuilder) BuildWithListener(listener net.Listener) *TestContext {
	environment := tcb.Environment(tcb.envPreHooks...)

	for _, envPostHook := range tcb.envPostHooks {
		envPostHook(environment, tcb.Servers)
	}
	wg := &sync.WaitGroup{}

	smServer, smRepository := newSMServer(environment, wg, tcb.smExtensions, listener)
	tcb.Servers[SMServer] = smServer

	SM := httpexpect.New(ginkgo.GinkgoT(), smServer.URL())
	oauthServer := tcb.Servers[OauthServer].(*OAuthServer)
	accessToken := oauthServer.CreateToken(tcb.defaultTokenClaims)
	SMWithOAuth := SM.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+accessToken).WithClient(tcb.HttpClient)
	})

	tenantAccessToken := oauthServer.CreateToken(tcb.tenantTokenClaims)
	SMWithOAuthForTenant := SM.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+tenantAccessToken).WithClient(tcb.HttpClient)
	})

	testContext := &TestContext{
		wg:                   wg,
		SM:                   &SMExpect{SM},
		SMWithOAuth:          &SMExpect{SMWithOAuth},
		SMWithOAuthForTenant: &SMExpect{SMWithOAuthForTenant},
		Servers:              tcb.Servers,
		SMRepository:         smRepository,
	}

	RemoveAllOperations(testContext.SMRepository)
	RemoveAllInstances(testContext.SMRepository)
	RemoveAllBrokers(testContext.SMWithOAuth)
	RemoveAllPlatforms(testContext.SMWithOAuth)

	if !tcb.shouldSkipBasicAuthClient {
		platformJSON := MakePlatform("tcb-platform-test", "tcb-platform-test", "platform-type", "test-platform")
		platform := RegisterPlatformInSM(platformJSON, testContext.SMWithOAuth, map[string]string{})
		SMWithBasic := SM.Builder(func(req *httpexpect.Request) {
			username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
			req.WithBasicAuth(username, password).WithClient(tcb.HttpClient)
		})
		testContext.SMWithBasic = &SMExpect{SMWithBasic}
		testContext.TestPlatform = platform
	}

	return testContext
}

func NewSMListener() (net.Listener, error) {
	minPort := 8100
	maxPort := 9999
	retries := 10

	var listener net.Listener
	var err error
	for ; retries >= 0; retries-- {
		rand.Seed(time.Now().UnixNano())
		port := rand.Intn(maxPort-minPort) + minPort

		smURL := "127.0.0.1:" + strconv.Itoa(port)
		listener, err = net.Listen("tcp", smURL)
		if err == nil {
			return listener, nil
		}
	}

	return nil, fmt.Errorf("unable to create sm listener: %s", err)
}

func newSMServer(smEnv env.Environment, wg *sync.WaitGroup, fs []func(ctx context.Context, smb *sm.ServiceManagerBuilder, env env.Environment) error, listener net.Listener) (*testSMServer, storage.TransactionalRepository) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.New(smEnv)
	if err != nil {
		panic(err)
	}

	smb, err := sm.New(ctx, cancel, smEnv, cfg)
	if err != nil {
		panic(err)
	}

	for _, registerExtensionsFunc := range fs {
		if err := registerExtensionsFunc(ctx, smb, smEnv); err != nil {
			panic(fmt.Sprintf("error creating test SM server: %s", err))
		}
	}
	serviceManager := smb.Build()

	err = smb.Notificator.Start(ctx, wg)
	if err != nil {
		panic(err)
	}
	err = smb.NotificationCleaner.Start(ctx, wg)
	if err != nil {
		panic(err)
	}

	testServer := httptest.NewUnstartedServer(serviceManager.Server.Router)
	if listener != nil {
		testServer.Listener.Close()
		testServer.Listener = listener
	}
	testServer.Start()

	return &testSMServer{
		cancel: cancel,
		Server: testServer,
	}, smb.Storage
}

func (ctx *TestContext) RegisterBrokerWithCatalogAndLabels(catalog SBCatalog, brokerData Object) (string, Object, *BrokerServer) {
	return ctx.RegisterBrokerWithCatalogAndLabelsExpect(catalog, brokerData, ctx.SMWithOAuth)
}

func (ctx *TestContext) RegisterBrokerWithCatalogAndLabelsExpect(catalog SBCatalog, brokerData Object, expect *SMExpect) (string, Object, *BrokerServer) {
	brokerServer := NewBrokerServerWithCatalog(catalog)
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	UUID2, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	brokerJSON := Object{
		"name":        UUID.String(),
		"broker_url":  brokerServer.URL(),
		"description": UUID2.String(),
		"credentials": Object{
			"basic": Object{
				"username": brokerServer.Username,
				"password": brokerServer.Password,
			},
		},
	}

	MergeObjects(brokerJSON, brokerData)

	broker := RegisterBrokerInSM(brokerJSON, expect, map[string]string{})
	brokerID := broker["id"].(string)
	brokerServer.ResetCallHistory()
	ctx.Servers[BrokerServerPrefix+brokerID] = brokerServer
	brokerJSON["id"] = brokerID
	return brokerID, broker, brokerServer
}

func MergeObjects(target, source Object) {
	for k, v := range source {
		obj, ok := v.(Object)
		if ok {
			var tobj Object
			tv, exists := target[k]
			if exists {
				tobj, ok = tv.(Object)
				if !ok {
					// incompatible types, just overwrite
					target[k] = v
					continue
				}
			} else {
				tobj = Object{}
				target[k] = tobj
			}
			MergeObjects(tobj, obj)
		} else {
			target[k] = v
		}
	}
}

func (ctx *TestContext) RegisterBrokerWithCatalog(catalog SBCatalog) (string, Object, *BrokerServer) {
	return ctx.RegisterBrokerWithCatalogAndLabels(catalog, Object{})
}

func (ctx *TestContext) RegisterBroker() (string, Object, *BrokerServer) {
	return ctx.RegisterBrokerWithCatalog(NewRandomSBCatalog())
}

func (ctx *TestContext) RegisterPlatform() *types.Platform {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	platformJSON := Object{
		"name":        UUID.String(),
		"type":        "testType",
		"description": "testDescrption",
	}
	return RegisterPlatformInSM(platformJSON, ctx.SMWithOAuth, map[string]string{})
}

func (ctx *TestContext) CleanupBroker(id string) {
	broker := ctx.Servers[BrokerServerPrefix+id]
	ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + id).Expect()
	broker.Close()
	delete(ctx.Servers, BrokerServerPrefix+id)
}

func (ctx *TestContext) Cleanup() {
	if ctx == nil {
		return
	}

	ctx.CleanupAdditionalResources()

	for _, server := range ctx.Servers {
		server.Close()
	}
	ctx.Servers = map[string]FakeServer{}

	ctx.wg.Wait()
}

func (ctx *TestContext) CleanupAdditionalResources() {
	if ctx == nil {
		return
	}

	if err := RemoveAllNotifications(ctx.SMRepository); err != nil && err != util.ErrNotFoundInStorage {
		panic(err)
	}
	if err := RemoveAllInstances(ctx.SMRepository); err != nil && err != util.ErrNotFoundInStorage {
		panic(err)
	}

	if err := RemoveAllOperations(ctx.SMRepository); err != nil && err != util.ErrNotFoundInStorage {
		panic(err)
	}

	ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL).Expect()

	if ctx.TestPlatform != nil {
		ctx.SMWithOAuth.DELETE(web.PlatformsURL).WithQuery("fieldQuery", fmt.Sprintf("id ne '%s'", ctx.TestPlatform.ID)).Expect()
	} else {
		ctx.SMWithOAuth.DELETE(web.PlatformsURL).Expect()
	}
	var smServer FakeServer
	for serverName, server := range ctx.Servers {
		if serverName == SMServer {
			smServer = server
		} else {
			server.Close()
		}
	}
	ctx.Servers = map[string]FakeServer{SMServer: smServer}

	for _, conn := range ctx.wsConnections {
		conn.Close()
	}
	ctx.wsConnections = nil
}

func (ctx *TestContext) ConnectWebSocket(platform *types.Platform, queryParams map[string]string) (*websocket.Conn, *http.Response, error) {
	smURL := ctx.Servers[SMServer].URL()
	smEndpoint, _ := url.Parse(smURL)
	smEndpoint.Scheme = "ws"
	smEndpoint.Path = web.NotificationsURL
	q := smEndpoint.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	smEndpoint.RawQuery = q.Encode()

	headers := http.Header{}
	encodedPlatform := base64.StdEncoding.EncodeToString([]byte(platform.Credentials.Basic.Username + ":" + platform.Credentials.Basic.Password))
	headers.Add("Authorization", "Basic "+encodedPlatform)

	wsEndpoint := smEndpoint.String()
	conn, resp, err := websocket.DefaultDialer.Dial(wsEndpoint, headers)
	if conn != nil {
		ctx.wsConnections = append(ctx.wsConnections, conn)
	}
	return conn, resp, err
}

func (ctx *TestContext) CloseWebSocket(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	conn.Close()
	for i, c := range ctx.wsConnections {
		if c == conn {
			ctx.wsConnections = append(ctx.wsConnections[:i], ctx.wsConnections[i+1:]...)
			return
		}
	}
}
