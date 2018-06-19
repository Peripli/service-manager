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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"

	"github.com/Peripli/service-manager/rest"
	"github.com/gavv/httpexpect"
)

func NewTestContext(api *rest.API) *TestContext {
	smServer := httptest.NewServer(GetServerRouter(api))
	SM := httpexpect.New(GinkgoT(), smServer.URL)

	RemoveAllBrokers(SM)
	RemoveAllPlatforms(SM)
	broker := &Broker{}
	brokerServer := httptest.NewServer(broker)
	brokerJSON := MakeBroker("broker1", brokerServer.URL, "")
	brokerID := RegisterBroker(brokerJSON, SM)
	osbURL := "/v1/osb/" + brokerID

	return &TestContext{
		SM:           SM,
		SMServer:     smServer,
		BrokerServer: brokerServer,
		OSBURL:       osbURL,
		Broker:       broker,
	}
}

type TestContext struct {
	SM                     *httpexpect.Expect
	SMServer, BrokerServer *httptest.Server
	OSBURL                 string
	Broker                 *Broker
}

func (ctx *TestContext) Cleanup() {
	if ctx == nil {
		return
	}
	if ctx.SMServer != nil {
		RemoveAllBrokers(ctx.SM)
		RemoveAllPlatforms(ctx.SM)
		ctx.SMServer.Close()
	}
	if ctx.BrokerServer != nil {
		ctx.BrokerServer.Close()
	}
}

type Broker struct {
	StatusCode     int
	ResponseBody   []byte
	Request        *http.Request
	RequestBody    *httpexpect.Value
	RawRequestBody []byte
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
