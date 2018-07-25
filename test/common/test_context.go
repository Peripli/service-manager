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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
)

var serviceCatalog = `{
	"services": [{
		"id": "1234",
		"name": "service1",
		"description": "sample-test",
		"bindable": true,
		"plans": [{
			"id": "plan-id",
			"name": "plan-name",
			"description": "plan-desc"
		}]
	}]
}`

func NewTestContext(additionalAPIs ...*web.API) *TestContext {
	ctx, cancel := context.WithCancel(context.Background())
	mockOauthServer := SetupMockOAuthServer()

	set := env.EmptyFlagSet()
	config.AddPFlags(set)
	set.Set("file.location", "./test/common")
	set.Set("api.token_issuer_url", mockOauthServer.URL)

	env, err := env.New(set)
	if err != nil {
		panic(err)
	}

	smanagerBuilder := sm.New(ctx, cancel, env)
	for _, additionalAPI := range additionalAPIs {
		smanagerBuilder.RegisterControllers(additionalAPI.Controllers...)
		smanagerBuilder.RegisterFilters(additionalAPI.Filters...)
	}
	serviceManager := smanagerBuilder.Build()
	smServer := httptest.NewServer(serviceManager.Server.Router)

	SM := httpexpect.New(GinkgoT(), smServer.URL)

	accessToken := RequestToken(mockOauthServer.URL)
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
		SMServer:    smServer,
		Brokers:     make(map[string]*Broker),
	}
}

type TestContext struct {
	SM          *httpexpect.Expect
	SMWithOAuth *httpexpect.Expect
	SMWithBasic *httpexpect.Expect
	SMServer    *httptest.Server

	Brokers map[string]*Broker
}

func (ctx *TestContext) RegisterBroker(name string, server *httptest.Server) *Broker {
	broker := &Broker{}
	if server == nil {
		server = httptest.NewServer(broker)
	}
	brokerJSON := MakeBroker(name, server.URL, "")
	broker.ResponseBody = []byte(serviceCatalog)
	brokerID := RegisterBroker(brokerJSON, ctx.SMWithOAuth)

	broker.OSBURL = "/v1/osb/" + brokerID
	broker.Server = server

	broker.ResponseBody = nil
	broker.Request = nil

	ctx.Brokers[name] = broker
	return broker
}

func (ctx *TestContext) Cleanup() {
	if ctx == nil {
		return
	}

	if ctx.SMServer != nil {
		RemoveAllBrokers(ctx.SMWithOAuth)
		RemoveAllPlatforms(ctx.SMWithOAuth)
		ctx.SMServer.Close()
	}

	for _, broker := range ctx.Brokers {
		if broker.Server != nil {
			broker.Server.Close()
		}
	}
}

type Broker struct {
	StatusCode     int
	ResponseBody   []byte
	Request        *http.Request
	RequestBody    *httpexpect.Value
	RawRequestBody []byte
	OSBURL         string
	Server         *httptest.Server
}

func (b *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.Request = req

	if req.Method == http.MethodPatch || req.Method == http.MethodPost || req.Method == http.MethodPut {
		var err error
		b.RawRequestBody, err = ioutil.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}
		var reqData interface{}
		err = json.Unmarshal(b.RawRequestBody, &reqData)
		if err != nil {
			panic(err)
		}

		b.RequestBody = httpexpect.NewValue(GinkgoT(), reqData)
	}

	code := b.StatusCode
	if code == 0 {
		code = http.StatusOK
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)

	rw.Write(b.ResponseBody)
}

func (b *Broker) Called() bool {
	return b.Request != nil
}
