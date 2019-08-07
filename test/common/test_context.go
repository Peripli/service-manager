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
	"strings"
	"sync"
	"time"

	"github.com/gavv/httpexpect"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/onsi/ginkgo"
	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/log"
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

	SM          *httpexpect.Expect
	SMWithOAuth *httpexpect.Expect
	// Requests a token the the "multitenant" oauth client - then token issued by this client contains
	// the "multitenant" client id behind the specified token claim in the api config
	// the token also contains a "tenant identifier" behind the configured tenant_indentifier claim that
	// will be compared with the value of the label specified in the "label key" configuration
	// In the end requesting brokers with this
	SMWithOAuthForTenant *httpexpect.Expect
	SMWithBasic          *httpexpect.Expect
	SMRepository         storage.Repository

	TestPlatform *types.Platform

	Servers               map[string]FakeServer
	Brokers               map[string]*BrokerServer
	PermanentBrokers      map[string]*BrokerServer
	PermanentPlatforms    map[string]*types.Platform
	PermanentVisibilities map[string]*types.Visibility
	BrokerFactory         *BrokerFactory
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

	testEnv, _ := env.Default(append([]func(set *pflag.FlagSet){config.AddPFlags}, additionalFlagFuncs...)...)
	return testEnv
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

	RemoveAllBrokers(SMWithOAuth)
	RemoveAllPlatforms(SMWithOAuth)

	testContext := &TestContext{
		wg:                    wg,
		SM:                    SM,
		SMWithOAuth:           SMWithOAuth,
		SMWithOAuthForTenant:  SMWithOAuthForTenant,
		Servers:               tcb.Servers,
		SMRepository:          smRepository,
		wsConnections:         make([]*websocket.Conn, 0),
		Brokers:               make(map[string]*BrokerServer),
		PermanentBrokers:      make(map[string]*BrokerServer),
		PermanentPlatforms:    make(map[string]*types.Platform),
		PermanentVisibilities: make(map[string]*types.Visibility),
		BrokerFactory:         NewBrokerFactory(),
	}

	if !tcb.shouldSkipBasicAuthClient {
		platformJSON := MakePlatform("tcb-platform-test", "tcb-platform-test", "platform-type", "test-platform")
		platform := RegisterPlatformInSM(platformJSON, SMWithOAuth, map[string]string{})
		SMWithBasic := SM.Builder(func(req *httpexpect.Request) {
			username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
			req.WithBasicAuth(username, password).WithClient(tcb.HttpClient)
		})
		testContext.SMWithBasic = SMWithBasic
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

func newSMServer(smEnv env.Environment, wg *sync.WaitGroup, fs []func(ctx context.Context, smb *sm.ServiceManagerBuilder, env env.Environment) error, listener net.Listener) (*testSMServer, storage.Repository) {
	ctx, cancel := context.WithCancel(context.Background())
	s := struct {
		Log *log.Settings
	}{
		Log: &log.Settings{
			Output: ginkgo.GinkgoWriter,
		},
	}
	err := smEnv.Unmarshal(&s)
	if err != nil {
		panic(err)
	}
	ctx = log.Configure(ctx, s.Log)

	cfg, err := config.NewForEnv(smEnv)
	if err != nil {
		panic(err)
	}

	smb, err := sm.New(ctx, cancel, cfg)
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
		_ = testServer.Listener.Close()
		testServer.Listener = listener
	}
	testServer.Start()

	return &testSMServer{
		cancel: cancel,
		Server: testServer,
	}, smb.Storage
}

// NewBrokerServer creates a new broker server with a random catalog. The server is managed by the test context.
func (ctx *TestContext) NewBrokerServer() *BrokerServer {
	return ctx.BrokerFactory.NewBrokerServer()
}

// NewBrokerServerWithCatalog creates a new broker server with a given catalog. The server is managed by the test context.
func (ctx *TestContext) NewBrokerServerWithCatalog(catalog SBCatalog) *BrokerServer {
	return ctx.BrokerFactory.NewBrokerServerWithCatalog(catalog)
}

// RegisterBroker registers a new broker with a given catalog and broker data in SM
func (ctx *TestContext) RegisterBrokerWithCatalogAndLabels(catalog SBCatalog, brokerData Object) (string, Object, *BrokerServer) {
	return ctx.RegisterBrokerWithCatalogAndLabelsExpect(catalog, brokerData, ctx.SMWithOAuth)
}

// RegisterBroker registers a new broker with a given catalog and broker data in SM which should not be deleted after each test execution
func (ctx *TestContext) RegisterPermanentBrokerWithCatalogAndLabels(catalog SBCatalog, brokerData Object) (string, Object, *BrokerServer) {
	return ctx.RegisterPermanentBrokerWithCatalogAndLabelsExpect(catalog, brokerData, ctx.SMWithOAuth)
}

// RegisterBroker registers a new broker with a given catalog, broker data and expect object in SM
func (ctx *TestContext) RegisterBrokerWithCatalogAndLabelsExpect(catalog SBCatalog, brokerData Object, expect *httpexpect.Expect) (string, Object, *BrokerServer) {
	return ctx.registerBrokerWithCatalogAndLabelsExpect(catalog, brokerData, expect, false)
}

// RegisterBroker registers a new broker with a given catalog, broker data and expect object in SM which should not be deleted after each test execution
func (ctx *TestContext) RegisterPermanentBrokerWithCatalogAndLabelsExpect(catalog SBCatalog, brokerData Object, expect *httpexpect.Expect) (string, Object, *BrokerServer) {
	return ctx.registerBrokerWithCatalogAndLabelsExpect(catalog, brokerData, expect, true)
}

// RegisterBrokerWithCatalog registers a new broker with a given catalog in SM
func (ctx *TestContext) RegisterBrokerWithCatalog(catalog SBCatalog) (string, Object, *BrokerServer) {
	return ctx.RegisterBrokerWithCatalogAndLabels(catalog, Object{})
}

// RegisterBroker registers a new broker with a given catalog in SM which should not be deleted after each test execution
func (ctx *TestContext) RegisterPermanentBrokerWithCatalog(catalog SBCatalog) (string, Object, *BrokerServer) {
	return ctx.RegisterPermanentBrokerWithCatalogAndLabels(catalog, Object{})
}

// RegisterBroker registers a new broker in SM
func (ctx *TestContext) RegisterBroker() (string, Object, *BrokerServer) {
	return ctx.RegisterBrokerWithCatalog(NewRandomSBCatalog())
}

// RegisterBroker registers a new broker in SM which should not be deleted after each test execution
func (ctx *TestContext) RegisterPermanentBroker() (string, Object, *BrokerServer) {
	return ctx.RegisterPermanentBrokerWithCatalog(NewRandomSBCatalog())
}

// RegisterVisibility registers a new visibility in SM
func (ctx *TestContext) RegisterVisibility(visibilityJSON Object) *types.Visibility {
	return RegisterVisibilityInSM(visibilityJSON, ctx.SMWithOAuth, map[string]string{})
}

// RegisterPermanentVisibility registers a new visibility in SM which should not be deleted after each test execution
func (ctx *TestContext) RegisterPermanentVisibility(visibilityJSON Object) *types.Visibility {
	visibility := ctx.RegisterVisibility(visibilityJSON)
	ctx.PermanentVisibilities[visibility.ID] = visibility
	return visibility
}

func (ctx *TestContext) registerBrokerWithCatalogAndLabelsExpect(catalog SBCatalog, brokerData Object, expect *httpexpect.Expect, isPermanent bool) (string, Object, *BrokerServer) {
	broker := ctx.NewBrokerServerWithCatalog(catalog)
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
		"broker_url":  broker.URL(),
		"description": UUID2.String(),
		"credentials": Object{
			"basic": Object{
				"username": broker.Username,
				"password": broker.Password,
			},
		},
	}

	MergeObjects(brokerJSON, brokerData)

	smBroker := RegisterBrokerInSM(brokerJSON, expect, map[string]string{})
	brokerID := smBroker["id"].(string)
	if isPermanent {
		ctx.PermanentBrokers[brokerID] = broker
	} else {
		ctx.Brokers[brokerID] = broker
	}

	broker.ResetCallHistory()
	brokerJSON["id"] = brokerID
	return brokerID, smBroker, broker
}

func (ctx *TestContext) registerPlatform(isPermanent bool, expect *httpexpect.Expect) *types.Platform {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	platformJSON := Object{
		"name":        UUID.String(),
		"type":        "testType",
		"description": "testDescription",
	}
	result := RegisterPlatformInSM(platformJSON, expect, map[string]string{})
	if isPermanent {
		ctx.PermanentPlatforms[result.ID] = result
	}
	return result
}

// RegisterPlatform registers a new platform in SM
func (ctx *TestContext) RegisterPlatform() *types.Platform {
	return ctx.registerPlatform(false, ctx.SMWithOAuth)
}

// RegisterPlatform registers a new platform in SM with a given expect object which will should not be deleted after each test execution
func (ctx *TestContext) RegisterPermanentPlatform() *types.Platform {
	return ctx.registerPlatform(true, ctx.SMWithOAuth)
}

// RegisterPlatform registers a new platform in SM with a given expect object
func (ctx *TestContext) RegisterPlatformExpect(expect *httpexpect.Expect) *types.Platform {
	return ctx.registerPlatform(false, expect)
}

// RegisterPlatform registers a new platform in SM which will should not be deleted after each test execution
func (ctx *TestContext) RegisterPermanentPlatformExpect(expect *httpexpect.Expect) *types.Platform {
	return ctx.registerPlatform(true, expect)
}

// CleanupAfterSuite removes all resources from the SM database. To be called only from AfterSuite.
// After this call test context must not be used.
func (ctx *TestContext) CleanupAfterSuite() {
	if ctx == nil {
		return
	}
	ctx.CleanupAllBrokers() // removes also all visibilities
	ctx.CleanupAllPlatforms()
	ctx.CleanupNotifications()
	ctx.CleanupWebsocketConnections()

	ctx.BrokerFactory.Close()
	ctx.CleanupFakeServers()
	ctx.wg.Wait() // Wait for SM server to finish
}

// CleanupAfterEach removes all resources that are not permanent. To be called in AfterEach
func (ctx *TestContext) CleanupAfterEach() {
	if ctx == nil {
		return
	}
	ctx.CleanupNotifications()
	ctx.CleanupBrokers()
	ctx.CleanupVisibilities()
	ctx.CleanupPlatforms()
	ctx.CleanupWebsocketConnections()
}

func (ctx *TestContext) CleanupNotifications() {
	_, err := ctx.SMRepository.Delete(context.TODO(), types.NotificationType)
	if err != nil && err != util.ErrNotFoundInStorage {
		panic(err)
	}
}

// CleanupBroker removes a broker from the SM and closes the server if available
func (ctx *TestContext) CleanupBroker(id string) {
	broker, ok := ctx.Brokers[id]
	if ok {
		delete(ctx.Brokers, id)
	} else {
		broker = ctx.PermanentBrokers[id]
		delete(ctx.PermanentBrokers, id)
	}
	ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + id).Expect()
	broker.Close()
}

func withIDNotInQuery(expect *httpexpect.Request, ids []string) *httpexpect.Request {
	fieldQueryRightOperand := "[" + strings.Join(ids, "||") + "]"
	expect.WithQuery("fieldQuery", "id notin "+fieldQueryRightOperand)
	return expect
}

// CleanupAllBrokers removes all service brokers registered in SM except the permanent ones
func (ctx *TestContext) CleanupBrokers() {
	if len(ctx.PermanentBrokers) == 0 {
		RemoveAllBrokers(ctx.SMWithOAuth)
	} else {
		permanentBrokers := make([]string, 0, len(ctx.PermanentBrokers))
		for brokerID := range ctx.PermanentBrokers {
			permanentBrokers = append(permanentBrokers, brokerID)
		}
		withIDNotInQuery(ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL), permanentBrokers).Expect()
	}
	for _, broker := range ctx.Brokers {
		broker.Close()
	}
	ctx.Brokers = make(map[string]*BrokerServer)
}

// CleanupAllBrokers removes all service brokers registered in SM
func (ctx *TestContext) CleanupAllBrokers() {
	RemoveAllBrokers(ctx.SMWithOAuth)
	for _, broker := range ctx.Brokers {
		broker.Close()
	}
	for _, broker := range ctx.PermanentBrokers {
		broker.Close()
	}
	ctx.Brokers = make(map[string]*BrokerServer)
	ctx.PermanentBrokers = make(map[string]*BrokerServer)
}

// CleanupPlatform removes a platform from the SM
func (ctx *TestContext) CleanupPlatform(id string) {
	ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + id).Expect()
	delete(ctx.PermanentPlatforms, id)
}

// CleanupPlatforms removes all registered platforms in SM except the permanent ones
func (ctx *TestContext) CleanupPlatforms() {
	permanentPlatformIDs := make([]string, 0, len(ctx.PermanentPlatforms)+1)
	if ctx.TestPlatform != nil {
		permanentPlatformIDs = append(permanentPlatformIDs, ctx.TestPlatform.ID)
	}
	for platformID := range ctx.PermanentPlatforms {
		permanentPlatformIDs = append(permanentPlatformIDs, platformID)
	}
	if len(permanentPlatformIDs) == 0 {
		RemoveAllPlatforms(ctx.SMWithOAuth)
	} else {
		withIDNotInQuery(ctx.SMWithOAuth.DELETE(web.PlatformsURL), permanentPlatformIDs).Expect()
	}
}

// CleanupAllPlatforms removes all registered platforms in SM
func (ctx *TestContext) CleanupAllPlatforms() {
	RemoveAllPlatforms(ctx.SMWithOAuth)
	ctx.PermanentPlatforms = make(map[string]*types.Platform)
}

// CleanupVisibilities removes all registered visibilities in SM except the permanent ones
func (ctx *TestContext) CleanupVisibilities() {
	if len(ctx.PermanentVisibilities) == 0 {
		RemoveAllVisibilities(ctx.SMWithOAuth)
		return
	}
	permanentVisibilitiesIDs := make([]string, 0, len(ctx.PermanentVisibilities))
	for visibilityID := range ctx.PermanentVisibilities {
		permanentVisibilitiesIDs = append(permanentVisibilitiesIDs, visibilityID)
	}
	withIDNotInQuery(ctx.SMWithOAuth.DELETE(web.VisibilitiesURL), permanentVisibilitiesIDs).Expect()
}

// CleanupVisibility removes a visibility from the SM
func (ctx *TestContext) CleanupVisibility(id string) {
	ctx.SMWithOAuth.DELETE(web.VisibilitiesURL + "/" + id).Expect()
	delete(ctx.PermanentVisibilities, id)
}

// CleanupFakeServers stops all servers including the SM
func (ctx *TestContext) CleanupFakeServers() {
	for _, server := range ctx.Servers {
		server.Close()
	}
	ctx.Servers = map[string]FakeServer{}
}

func (ctx *TestContext) CleanupWebsocketConnections() {
	for _, conn := range ctx.wsConnections {
		_ = conn.Close()
	}
	ctx.wsConnections = make([]*websocket.Conn, 0)
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
