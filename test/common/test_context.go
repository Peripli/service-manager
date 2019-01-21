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

var (
	e          env.Environment
	_, b, _, _ = runtime.Caller(0)
	basePath   = path.Dir(b)
)

type FlagValue struct {
	pflagValue pflag.Value
}

func (f FlagValue) Set(s string) error {
	return f.pflagValue.Set(s)
}

func (f FlagValue) String() string {
	return f.pflagValue.String()
}

func init() {
	e = TestEnv()
}

func SetTestFileLocation(set *pflag.FlagSet) {
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

	// will be used as default value and overridden if --file.location={{location}} is passed to go test
	additionalFlagFuncs = append(additionalFlagFuncs, SetTestFileLocation)

	return sm.DefaultEnv(additionalFlagFuncs...)
}

type ContextParams struct {
	RegisterExtensions func(smb *sm.ServiceManagerBuilder)
	DefaultTokenClaims map[string]interface{}
	Env                env.Environment
}

func NewSMServer(params *ContextParams, issuerURL string) *httptest.Server {
	var smEnv env.Environment
	if params.Env != nil {
		smEnv = params.Env
	} else {
		smEnv = e
	}

	smEnv.Set("api.token_issuer_url", issuerURL)

	flag.VisitAll(func(flag *flag.Flag) {
		if flag.Value.String() != "" {
			// if any of the go test flags have been set, propagate the value in sm env with highest prio
			// when env exposes the pflagset it would be better to instead override the pflag value instead
			smEnv.Set(flag.Name, flag.Value.String())
		}
	})

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
	smanagerBuilder := sm.New(ctx, cancel, smEnv)
	if params.RegisterExtensions != nil {
		params.RegisterExtensions(smanagerBuilder)
	}
	serviceManager := smanagerBuilder.Build()
	return httptest.NewServer(serviceManager.Server.Router)
}

func NewTestContext(params *ContextParams) *TestContext {
	if params == nil {
		params = &ContextParams{}
	}

	oauthServer := NewOAuthServer()
	oauthServer.Start()

	smServer := NewSMServer(params, oauthServer.URL)
	SM := httpexpect.New(GinkgoT(), smServer.URL)

	accessToken := oauthServer.CreateToken(params.DefaultTokenClaims)
	SMWithOAuth := SM.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+accessToken)
	})
	RemoveAllBrokers(SMWithOAuth)
	RemoveAllPlatforms(SMWithOAuth)

	platformJSON := MakePlatform("ctx-platform-test", "ctx-platform-test", "platform-type", "test-platform")
	platform := RegisterPlatformInSM(platformJSON, SMWithOAuth)
	SMWithBasic := SM.Builder(func(req *httpexpect.Request) {
		username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
		req.WithBasicAuth(username, password)
	})

	return &TestContext{
		SM:           SM,
		SMWithOAuth:  SMWithOAuth,
		SMWithBasic:  SMWithBasic,
		TestPlatform: platform,
		smServer:     smServer,
		OAuthServer:  oauthServer,
		brokers:      make(map[string]*BrokerServer),
	}
}

type TestContext struct {
	SM           *httpexpect.Expect
	SMWithOAuth  *httpexpect.Expect
	SMWithBasic  *httpexpect.Expect
	TestPlatform *types.Platform
	smServer     *httptest.Server
	OAuthServer  *OAuthServer
	brokers      map[string]*BrokerServer
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
		"broker_url":  brokerServer.URL,
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
	ctx.brokers[brokerID] = brokerServer
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
	broker := ctx.brokers[id]
	ctx.SMWithOAuth.DELETE("/v1/service_brokers/" + id).Expect()
	if broker != nil && broker.Server != nil {
		broker.Server.Close()
	}
	delete(ctx.brokers, id)
}

func (ctx *TestContext) Cleanup() {
	if ctx == nil {
		return
	}

	if ctx.SMWithOAuth != nil {
		RemoveAllBrokers(ctx.SMWithOAuth)
		RemoveAllPlatforms(ctx.SMWithOAuth)
	}

	for _, broker := range ctx.brokers {
		if broker != nil && broker.Server != nil {
			broker.Server.Close()
		}
	}

	if ctx.smServer != nil {
		ctx.smServer.Close()
	}

	if ctx.OAuthServer != nil {
		ctx.OAuthServer.Close()
	}
}

func (ctx *TestContext) CleanupAdditionalResources() {
	ctx.SMWithOAuth.DELETE("/v1/service_brokers").
		Expect()
	ctx.SMWithOAuth.DELETE("/v1/platforms").WithQuery("fieldQuery", "id != "+ctx.TestPlatform.ID).
		Expect()
}
