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
	"flag"
	"fmt"
	"net/http/httptest"
	"path"
	"runtime"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/onsi/ginkgo"
	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
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

	shouldSkipBasicAuthClient bool

	Environment func(f ...func(set *pflag.FlagSet)) env.Environment
	Servers     map[string]FakeServer
}

type TestContext struct {
	SM           *httpexpect.Expect
	SMWithOAuth  *httpexpect.Expect
	SMWithBasic  *httpexpect.Expect
	TestPlatform *types.Platform

	Servers map[string]FakeServer
}

type testSMServer struct {
	*httptest.Server
}

func (ts *testSMServer) URL() string {
	return ts.Server.URL
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
		Servers: map[string]FakeServer{
			"oauth-server": NewOAuthServer(),
		},
	}
}

func SetTestFileLocation(set *pflag.FlagSet) {
	_, b, _, _ := runtime.Caller(0)
	basePath := path.Dir(b)
	set.Set("file.location", basePath)
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

	return sm.DefaultEnv(additionalFlagFuncs...)
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
	environment := tcb.Environment(tcb.envPreHooks...)

	for _, envPostHook := range tcb.envPostHooks {
		envPostHook(environment, tcb.Servers)
	}

	smServer := newSMServer(environment, tcb.smExtensions)
	tcb.Servers[SMServer] = smServer

	SM := httpexpect.New(GinkgoT(), smServer.URL())
	oauthServer := tcb.Servers[OauthServer].(*OAuthServer)
	accessToken := oauthServer.CreateToken(tcb.defaultTokenClaims)
	SMWithOAuth := SM.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+accessToken)
	})
	RemoveAllBrokers(SMWithOAuth)
	RemoveAllPlatforms(SMWithOAuth)

	testContext := &TestContext{
		SM:          SM,
		SMWithOAuth: SMWithOAuth,
		Servers:     tcb.Servers,
	}

	if !tcb.shouldSkipBasicAuthClient {
		platformJSON := MakePlatform("tcb-platform-test", "tcb-platform-test", "platform-type", "test-platform")
		platform := RegisterPlatformInSM(platformJSON, SMWithOAuth)
		SMWithBasic := SM.Builder(func(req *httpexpect.Request) {
			username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
			req.WithBasicAuth(username, password)
		})
		testContext.SMWithBasic = SMWithBasic
		testContext.TestPlatform = platform
	}

	return testContext

}

func newSMServer(smEnv env.Environment, fs []func(ctx context.Context, smb *sm.ServiceManagerBuilder, env env.Environment) error) *testSMServer {
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
	smb := sm.New(ctx, cancel, smEnv)
	for _, registerExtensionsFunc := range fs {
		if err := registerExtensionsFunc(ctx, smb, smEnv); err != nil {
			panic(fmt.Sprintf("error creating test SM server: %s", err))
		}
	}
	serviceManager := smb.Build()
	return &testSMServer{
		Server: httptest.NewServer(serviceManager.Server.Router),
	}
}

func (ctx *TestContext) RegisterBrokerWithCatalogAndLabels(catalog SBCatalog, labels Object) (string, Object, *BrokerServer) {
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

	if len(labels) != 0 {
		brokerJSON["labels"] = labels
	}

	brokerID := RegisterBrokerInSM(brokerJSON, ctx.SMWithOAuth)
	brokerServer.ResetCallHistory()
	ctx.Servers[BrokerServerPrefix+brokerID] = brokerServer
	brokerJSON["id"] = brokerID
	return brokerID, brokerJSON, brokerServer
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
	return RegisterPlatformInSM(platformJSON, ctx.SMWithOAuth)
}

func (ctx *TestContext) CleanupBroker(id string) {
	broker := ctx.Servers[BrokerServerPrefix+id]
	ctx.SMWithOAuth.DELETE("/v1/service_brokers/" + id).Expect()
	broker.Close()
	delete(ctx.Servers, BrokerServerPrefix+id)
}

func (ctx *TestContext) Cleanup() {
	RemoveAllBrokers(ctx.SMWithOAuth)
	RemoveAllPlatforms(ctx.SMWithOAuth)

	for _, server := range ctx.Servers {
		server.Close()
	}
}

func (ctx *TestContext) CleanupAdditionalResources() {
	ctx.SMWithOAuth.DELETE("/v1/service_brokers").
		Expect()
	ctx.SMWithOAuth.DELETE("/v1/platforms").WithQuery("fieldQuery", "id != "+ctx.TestPlatform.ID).
		Expect()
}
