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

	"github.com/mitchellh/mapstructure"

	"github.com/Peripli/service-manager/test/tls_settings"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/bcrypt"

	"github.com/Peripli/service-manager/operations"

	"github.com/Peripli/service-manager/api/extensions/security"

	"github.com/gavv/httpexpect"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/onsi/ginkgo"
	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

func init() {
	// dummy env to put SM pflags to flags
	TestEnv(SetTestFileLocation)
}

const SMServer = "sm-server"
const OauthServer = "oauth-server"
const TenantOauthServer = "tenant-oauth-server"
const BrokerServerPrefix = "broker-"
const BrokerServerPrefixTLS = "broker-tls-"

type TestContextBuilder struct {
	envPreHooks  []func(set *pflag.FlagSet)
	envPostHooks []func(env env.Environment, servers map[string]FakeServer)

	smExtensions       []func(ctx context.Context, smb *sm.ServiceManagerBuilder, env env.Environment) error
	defaultTokenClaims map[string]interface{}
	tenantTokenClaims  map[string]interface{}

	shouldSkipBasicAuthClient bool
	basicAuthPlatformName     string

	Environment func(f ...func(set *pflag.FlagSet)) env.Environment
	Servers     map[string]FakeServer
	HttpClient  *http.Client

	useSeparateOAuthServerForTenantAccess bool
}

type BrokerContext struct {
	BrokerServer     *BrokerServer
	JSON             Object
	ID               string
	GeneratedCatalog SBCatalog
}

type BrokerUtilsContext struct {
	value      *httpexpect.Array
	fieldValue string
	selected   *BrokerContext
}

type BrokerUtils struct {
	Broker        BrokerContext
	BrokerWithTLS BrokerContext
	authContext   *SMExpect
	Context       BrokerUtilsContext
}

func (ctx *BrokerUtils) Cleanup(broker BrokerContext) {
	ctx.authContext.DELETE(web.ServiceBrokersURL + "/" + broker.ID).Expect()
	broker.BrokerServer.Close()
}
func (ctx *BrokerUtils) GetBrokerAsParams() (string, Object, *BrokerServer) {
	return ctx.Broker.ID, ctx.Broker.JSON, ctx.Broker.BrokerServer
}

func (ctx *BrokerUtils) SelectBroker(broker *BrokerContext) *BrokerUtils {
	ctx.Context.selected = broker
	return ctx
}

func (ctx *BrokerUtils) GetPlanCatalogId(service, plan int) string {
	catalog := string(ctx.Context.selected.GeneratedCatalog)
	return gjson.Get(catalog, fmt.Sprintf("services.%d.plans.%d.id", service, plan)).Str
}

func (ctx *BrokerUtils) GetBrokerOSBURL(brokerID string) string {
	return ctx.Context.selected.BrokerServer.URL() + "/v1/osb/" + brokerID
}

func (ctx *BrokerUtils) GetServiceCatalogId(service int) string {
	smBrokerServiceIdPlan := string(ctx.Context.selected.GeneratedCatalog)
	return gjson.Get(smBrokerServiceIdPlan, fmt.Sprintf("services.%d.id", service)).Str
}

func (ctx *BrokerUtils) RegisterPlatformToBroker(username, password, brokerID string) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	payload := map[string]interface{}{
		"broker_id":       brokerID,
		"username":        username,
		"password_hash":   string(passwordHash),
		"notification_id": "",
	}

	ctx.authContext.Request(http.MethodPut, web.BrokerPlatformCredentialsURL).
		WithJSON(payload).Expect().Status(http.StatusOK)
}

func (ctx *BrokerUtils) SetAuthContext(authContext *SMExpect) *BrokerUtils {
	ctx.authContext = authContext
	return ctx
}
func (ctx *BrokerUtils) GetServiceOfferings(brokerId string) *BrokerUtils {
	ctx.Context.value = ctx.authContext.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerId))
	return ctx
}

func (ctx *BrokerUtils) AddPlanVisibilityForPlatform(planCatalogID string, platformID string, orgID string) *BrokerUtils {

	smPlanID := ctx.authContext.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("catalog_id eq '%s'", planCatalogID)).
		First().Object().Value("id").String().Raw()

	visibilityID := RegisterVisibilityForPlanAndPlatform(ctx.authContext, smPlanID, platformID)
	if orgID != "" {
		patchLabelsBody := make(map[string]interface{})
		patchLabels := []types.LabelChange{{
			Operation: types.AddLabelOperation,
			Key:       "organization_guid",
			Values:    []string{orgID},
		}}
		patchLabelsBody["labels"] = patchLabels

		ctx.authContext.PATCH(web.VisibilitiesURL + "/" + visibilityID).
			WithJSON(patchLabelsBody).
			Expect().
			Status(http.StatusOK)
	}

	return ctx
}

func (ctx *BrokerUtils) GetServicePlans(forOffering int, key string) *BrokerUtils {
	ctx.Context.value = ctx.authContext.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", ctx.Context.value.Element(forOffering).Object().Value(key).String().Raw()))
	return ctx
}

func (ctx *BrokerUtils) GetPlan(forPlan int, key string) *BrokerUtils {
	ctx.Context.fieldValue = ctx.Context.value.Element(1).Object().Value(key).String().Raw()
	return ctx
}

func (ctx *BrokerUtils) GetAsServiceInstancePayload() (Object, string) {
	ID, _ := uuid.NewV4()
	payload := Object{
		"name":             "test-instance" + ID.String(),
		"service_plan_id":  ctx.Get(),
		"maintenance_info": "{}",
	}

	return payload, ctx.Get()
}

func (ctx *BrokerUtils) Get() string {
	return ctx.Context.fieldValue
}

//ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", planId))

type TestContext struct {
	wg            *sync.WaitGroup
	wsConnections []*websocket.Conn

	Config      *config.Settings
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
	SMScheduler          *operations.Scheduler
	TestPlatform         *types.Platform
	TenantTokenProvider  func() string
	TestContextData      BrokerUtils
	Servers              map[string]FakeServer
	HttpClient           *http.Client
	Maintainer           *operations.Maintainer
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

func (expect *SMExpect) SetBasicCredentials(ctx *TestContext, username, password string) {
	expect.Expect = ctx.SM.Builder(func(req *httpexpect.Request) {

		req.WithBasicAuth(username, password).WithClient(ctx.HttpClient)
	})
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
	return NewTestContextBuilderWithSecurity().Build()
}

func NewTestContextBuilderWithSecurity() *TestContextBuilder {
	return NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
		cfg, err := config.New(e)
		if err != nil {
			return err
		}
		if err := security.Register(ctx, cfg, smb); err != nil {
			return err
		}

		return nil
	})
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
				env.Set("api.token_issuer_url", servers[OauthServer].URL())
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
		Servers:            map[string]FakeServer{},
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

	env, _ := env.DefaultLogger(context.TODO(), append([]func(set *pflag.FlagSet){config.AddPFlags}, additionalFlagFuncs...)...)
	return env
}

func (tcb *TestContextBuilder) WithBasicAuthPlatformName(name string) *TestContextBuilder {
	tcb.basicAuthPlatformName = name

	return tcb
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

func (tcb *TestContextBuilder) ShouldUseSeparateOAuthServerForTenantAccess(use bool) *TestContextBuilder {
	tcb.useSeparateOAuthServerForTenantAccess = use
	return tcb
}

func (tcb *TestContextBuilder) Build() *TestContext {
	return tcb.BuildWithListener(nil, true)
}

func (tcb *TestContextBuilder) BuildWithCleanup(shouldCleanup bool) *TestContext {
	return tcb.BuildWithListener(nil, shouldCleanup)
}

func (tcb *TestContextBuilder) BuildWithoutCleanup() *TestContext {
	return tcb.SkipBasicAuthClientSetup(true).BuildWithListener(nil, false)
}

func (tcb *TestContextBuilder) BuildWithListener(listener net.Listener, cleanup bool) *TestContext {
	environment := tcb.Environment(tcb.envPreHooks...)

	tcb.Servers[OauthServer] = NewOAuthServer()
	if tcb.useSeparateOAuthServerForTenantAccess {
		tcb.Servers[TenantOauthServer] = NewOAuthServer()
	} else {
		tcb.Servers[TenantOauthServer] = tcb.Servers[OauthServer]
	}

	for _, envPostHook := range tcb.envPostHooks {
		envPostHook(environment, tcb.Servers)
	}
	wg := &sync.WaitGroup{}

	smServer, smRepository, smScheduler, maintainer, config := newSMServer(environment, wg, tcb.smExtensions, listener)
	tcb.Servers[SMServer] = smServer

	SM := httpexpect.New(ginkgo.GinkgoT(), smServer.URL())
	oauthServer := tcb.Servers[OauthServer].(*OAuthServer)
	tenantOauthServer := tcb.Servers[TenantOauthServer].(*OAuthServer)
	accessToken := oauthServer.CreateToken(tcb.defaultTokenClaims)
	SMWithOAuth := SM.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+accessToken).WithClient(tcb.HttpClient)
	})

	tenantAccessToken := tenantOauthServer.CreateToken(tcb.tenantTokenClaims)
	SMWithOAuthForTenant := SM.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+tenantAccessToken).WithClient(tcb.HttpClient)
	})

	testContext := &TestContext{
		wg:                   wg,
		Config:               config,
		SM:                   &SMExpect{Expect: SM},
		Maintainer:           maintainer,
		SMWithOAuth:          &SMExpect{Expect: SMWithOAuth},
		SMWithOAuthForTenant: &SMExpect{Expect: SMWithOAuthForTenant},
		Servers:              tcb.Servers,
		SMRepository:         smRepository,
		SMScheduler:          smScheduler,
		HttpClient:           tcb.HttpClient,
		TenantTokenProvider: func() string {
			return tenantOauthServer.CreateToken(tcb.tenantTokenClaims)
		},
	}

	if cleanup {
		RemoveAllBindings(testContext)
		RemoveAllInstances(testContext)
		RemoveAllBrokers(testContext.SMRepository)
		RemoveAllPlatforms(testContext.SMRepository)
		RemoveAllOperations(testContext.SMRepository)
	}

	if !tcb.shouldSkipBasicAuthClient {
		platform, err := tcb.prepareTestPlatform(testContext.SMWithOAuth, smRepository)
		if err != nil {
			panic(err)
		}
		SMWithBasic := SM.Builder(func(req *httpexpect.Request) {
			username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
			req.WithBasicAuth(username, password).WithClient(tcb.HttpClient)
		})
		testContext.SMWithBasic = &SMExpect{SMWithBasic}
		testContext.TestPlatform = platform
	}

	return testContext
}

func (tcb *TestContextBuilder) prepareTestPlatform(smClient *SMExpect, repository storage.TransactionalRepository) (*types.Platform, error) {
	if tcb.basicAuthPlatformName == "" {
		tcb.basicAuthPlatformName = "basic-auth-default-test-platform"
		resp := smClient.GET(web.PlatformsURL + "/" + tcb.basicAuthPlatformName).Expect()
		if resp.Raw().StatusCode == http.StatusNotFound {
			platformJSON := MakePlatform(tcb.basicAuthPlatformName, tcb.basicAuthPlatformName, "platform-type", "test-platform")
			platform := RegisterPlatformInSM(platformJSON, smClient, map[string]string{})
			platform.Active = true
			platformObj, err := repository.Update(context.Background(), platform, nil)

			return platformObj.(*types.Platform), err
		}

		if resp.Raw().StatusCode != http.StatusOK {
			panic(resp.Raw().Status)
		}
		var platform types.Platform
		if err := mapstructure.Decode(resp.JSON().Object().Raw(), &platform); err != nil {
			return nil, err
		}

		return &platform, nil
	}

	smClient.DELETE(web.PlatformsURL + "/" + tcb.basicAuthPlatformName)
	platformJSON := MakePlatform(tcb.basicAuthPlatformName, tcb.basicAuthPlatformName, "platform-type", "test-platform")
	platform := RegisterPlatformInSM(platformJSON, smClient, map[string]string{})

	return platform, nil
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

func newSMServer(smEnv env.Environment, wg *sync.WaitGroup, fs []func(ctx context.Context, smb *sm.ServiceManagerBuilder, env env.Environment) error, listener net.Listener) (*testSMServer, storage.TransactionalRepository, *operations.Scheduler, *operations.Maintainer, *config.Settings) {
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
	scheduler := operations.NewScheduler(ctx, smb.Storage, cfg.Operations, 1000, wg)
	return &testSMServer{
		cancel: cancel,
		Server: testServer,
	}, smb.Storage, scheduler, smb.OperationMaintainer, cfg
}

func (ctx *TestContext) RegisterBrokerWithCatalogAndLabels(catalog SBCatalog, brokerData Object, expectedStatus int) *BrokerUtils {
	return ctx.RegisterBrokerWithCatalogAndLabelsExpect(catalog, brokerData, ctx.SMWithOAuth, expectedStatus)
}

func (ctx *TestContext) RegisterBrokerWithRandomCatalogAndTLS(expect *SMExpect) *BrokerUtils {
	generatedCatalog := NewRandomSBCatalog()
	brokerServerWithTLS := NewBrokerServerWithTLSAndCatalog(generatedCatalog, []byte(tls_settings.BrokerCertificate), []byte(tls_settings.BrokerCertificateKey),
		[]byte(tls_settings.ClientCaCertificate))

	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	UUID2, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	brokerJSONWithTLS := Object{
		"name":        BrokerServerPrefixTLS + UUID.String(),
		"broker_url":  brokerServerWithTLS.URL(),
		"description": BrokerServerPrefixTLS + UUID2.String(),
		"credentials": Object{
			"tls": Object{
				"client_certificate": tls_settings.ClientCertificate,
				"client_key":         tls_settings.ClientKey,
			},
		},
	}
	brokerTLS := RegisterBrokerInSM(brokerJSONWithTLS, expect, map[string]string{}, http.StatusCreated)
	tlsBrokerID := brokerTLS["id"].(string)
	brokerJSONWithTLS["id"] = tlsBrokerID

	brokerUtils := BrokerUtils{
		BrokerWithTLS: BrokerContext{
			BrokerServer:     brokerServerWithTLS,
			JSON:             brokerTLS,
			ID:               tlsBrokerID,
			GeneratedCatalog: generatedCatalog,
		},
	}
	ctx.Servers[BrokerServerPrefix+tlsBrokerID] = brokerServerWithTLS
	return &brokerUtils

}

func (ctx *TestContext) RegisterBrokerWithCatalogAndLabelsExpect(catalog SBCatalog, brokerData Object, expect *SMExpect, expectedStatus int) *BrokerUtils {
	brokerServer, brokerServerWithTLS, brokerJSON, broker := ctx.TryRegisterBrokerWithCatalogAndLabels(catalog, brokerData, expect, expectedStatus)
	brokerID := broker["id"].(string)

	brokerServer.ResetCallHistory()
	brokerServerWithTLS.ResetCallHistory()
	ctx.Servers[BrokerServerPrefix+brokerID] = brokerServer
	brokerJSON["id"] = brokerID

	brokerUtils := BrokerUtils{
		Broker: BrokerContext{
			BrokerServer: brokerServer,
			JSON:         broker,
			ID:           brokerID,
		},
	}

	return &brokerUtils
}

func (ctx *TestContext) TryRegisterBrokerWithCatalogAndLabels(catalog SBCatalog, brokerData Object, expect *SMExpect, expectedStatus int) (*BrokerServer, *BrokerServer, Object, Object) {
	brokerServer := NewBrokerServerWithCatalog(catalog)
	brokerServerWithTLS := NewBrokerServerMTLS([]byte(tls_settings.BrokerCertificate), []byte(tls_settings.BrokerCertificateKey), []byte(tls_settings.ClientCaCertificate))
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
	broker := RegisterBrokerInSM(brokerJSON, expect, map[string]string{}, expectedStatus)
	return brokerServer, brokerServerWithTLS, brokerJSON, broker
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

func (ctx *TestContext) RegisterBrokerWithCatalog(catalog SBCatalog) *BrokerUtils {
	return ctx.RegisterBrokerWithCatalogAndLabels(catalog, Object{}, http.StatusCreated)
}

func (ctx *TestContext) RegisterBroker() *BrokerUtils {
	return ctx.RegisterBrokerWithCatalog(NewRandomSBCatalog())
}

func (ctx *TestContext) RegisterPlatform() *types.Platform {
	return ctx.RegisterPlatformAndActivate(true)
}

func (ctx *TestContext) RegisterPlatformAndActivate(activate bool) *types.Platform {
	return ctx.RegisterPlatformWithTypeAndActivate("test-type", activate)
}

func (ctx *TestContext) RegisterTenantPlatform() *types.Platform {
	return RegisterPlatformInSM(GenerateRandomPlatform(), ctx.SMWithOAuthForTenant, map[string]string{})
}

func (ctx *TestContext) RegisterPlatformWithType(platformType string) *types.Platform {
	return ctx.RegisterPlatformWithTypeAndActivate(platformType, true)
}

func (ctx *TestContext) RegisterPlatformWithTypeAndActivate(platformType string, activate bool) *types.Platform {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	platformJSON := Object{
		"name":        UUID.String(),
		"type":        platformType,
		"description": "testDescrption",
	}
	platform := RegisterPlatformInSM(platformJSON, ctx.SMWithOAuth, map[string]string{})
	if activate {
		platform.Active = true
		platformObj, err := ctx.SMRepository.Update(context.Background(), platform, nil)
		if err != nil {
			panic(err)
		}
		platform = platformObj.(*types.Platform)
	}

	return platform
}

func (ctx *TestContext) NewTenantExpect(clientID, tenantIdentifier string, scopes ...string) *SMExpect {
	tenantOauthServer := ctx.Servers[TenantOauthServer].(*OAuthServer)

	accessToken := tenantOauthServer.CreateToken(map[string]interface{}{
		"cid":        clientID,
		"zid":        tenantIdentifier,
		"grant_type": "password",
		"scope":      scopes,
	})

	return &SMExpect{
		Expect: ctx.SM.Builder(func(req *httpexpect.Request) {
			req.WithHeader("Authorization", "Bearer "+accessToken)
		}),
	}
}

func (ctx *TestContext) CleanupBroker(id string) {
	broker := ctx.Servers[BrokerServerPrefix+id]
	ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + id).Expect()
	broker.Close()
	delete(ctx.Servers, BrokerServerPrefix+id)
}

func (ctx *TestContext) Cleanup() {
	ctx.CleanupAll(true)
}

func (ctx *TestContext) CleanupAll(cleanupResources bool) {
	if ctx == nil {
		return
	}

	if cleanupResources {
		ctx.CleanupAdditionalResources()
	}

	for _, server := range ctx.Servers {
		server.Close()
	}
	ctx.Servers = map[string]FakeServer{}

	ctx.wg.Wait()
}

func (ctx *TestContext) CleanupPlatforms() {
	if ctx.TestPlatform != nil {
		ctx.SMWithOAuth.DELETE(web.PlatformsURL).WithQuery("fieldQuery", fmt.Sprintf("id notin ('%s', '%s')", ctx.TestPlatform.ID, types.SMPlatform)).Expect()
	} else {
		ctx.SMWithOAuth.DELETE(web.PlatformsURL).WithQuery("fieldQuery", fmt.Sprintf("id ne '%s'", types.SMPlatform)).Expect()
	}
}

func (ctx *TestContext) CleanupAdditionalResources() {
	if ctx == nil {
		return
	}

	RemoveAllNotifications(ctx.SMRepository)
	RemoveAllBindings(ctx)
	RemoveAllInstances(ctx)
	RemoveAllOperations(ctx.SMRepository)

	ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL).Expect()

	ctx.CleanupPlatforms()
	serversToDelete := make([]string, 0)
	for serverName, server := range ctx.Servers {
		if serverName != SMServer && serverName != OauthServer && serverName != TenantOauthServer {
			serversToDelete = append(serversToDelete, serverName)
			server.Close()
		}
	}
	for _, sname := range serversToDelete {
		delete(ctx.Servers, sname)
	}

	for _, conn := range ctx.wsConnections {
		conn.Close()
	}
	ctx.wsConnections = nil
}

func (ctx *TestContext) ConnectWebSocket(platform *types.Platform,
	queryParams map[string]string,
	withHeaders map[string]string) (*websocket.Conn, *http.Response, error) {
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
	if withHeaders != nil {
		for k, v := range withHeaders {
			headers.Add(k, v)
		}
	}
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
