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
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
)

type ContextParams struct {
	Environment        env.Environment
	RegisterExtensions func(api *web.API)
	DefaultTokenClaims map[string]interface{}
}

func LoadEnvironment(confgiFileDir string) env.Environment {
	return sm.DefaultEnv(func(set *pflag.FlagSet) {
		set.Set("file.location", confgiFileDir)
	})
}

func buildSM(params *ContextParams, issuerURL string) *httptest.Server {
	if params.Environment == nil {
		params.Environment = LoadEnvironment("./test/common")
	}
	params.Environment.Set("api.token_issuer_url", issuerURL)

	ctx, cancel := context.WithCancel(context.Background())
	smanagerBuilder := sm.New(ctx, cancel, params.Environment)
	if params.RegisterExtensions != nil {
		params.RegisterExtensions(smanagerBuilder.API)
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

	smServer := buildSM(params, oauthServer.URL)
	SM := httpexpect.New(GinkgoT(), smServer.URL)

	accessToken := oauthServer.CreateToken(params.DefaultTokenClaims)
	SMWithOAuth := SM.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+accessToken)
	})

	RemoveAllBrokers(SMWithOAuth)
	RemoveAllPlatforms(SMWithOAuth)

	platformJSON := MakePlatform("ctx-platform-test", "ctx-platform-test", "platform-type", "test-platform")
	platform := RegisterPlatform(platformJSON, SMWithOAuth)
	SMWithBasic := SM.Builder(func(req *httpexpect.Request) {
		username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
		req.WithBasicAuth(username, password)
	})

	return &TestContext{
		SM:          SM,
		SMWithOAuth: SMWithOAuth,
		SMWithBasic: SMWithBasic,
		brokers:     make(map[string]*Broker),
		smServer:    smServer,
		OAuthServer: oauthServer,
	}
}

type TestContext struct {
	SM          *httpexpect.Expect
	SMWithOAuth *httpexpect.Expect
	SMWithBasic *httpexpect.Expect

	smServer    *httptest.Server
	OAuthServer *OAuthServer
	brokers     map[string]*Broker
}

func (ctx *TestContext) RegisterBroker(name string, server *httptest.Server) *Broker {
	broker := &Broker{}
	if server == nil {
		server = httptest.NewServer(broker)
	}
	brokerJSON := MakeBroker(name, server.URL, "")
	broker.ID = RegisterBroker(brokerJSON, ctx.SMWithOAuth)

	broker.OSBURL = "/v1/osb/" + broker.ID
	broker.Server = server

	broker.Request = nil

	ctx.brokers[name] = broker
	return broker
}

func (ctx *TestContext) CleanupBroker(name string) {
	broker := ctx.brokers[name]
	ctx.SMWithOAuth.DELETE("/v1/service_brokers/" + broker.ID).Expect()
	if broker.Server != nil {
		broker.Server.Close()
	}
	delete(ctx.brokers, name)
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
		if broker.Server != nil {
			broker.Server.Close()
		}
	}

	if ctx.smServer != nil {
		ctx.smServer.Close()
	}
	ctx.OAuthServer.Close()
}
