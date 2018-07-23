/*
 * Copyright 2018 The Service Manager Authors
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

package api

import (
	"strings"
	"testing"

	"net/http"
	"net/http/httptest"

	"bytes"
	"fmt"

	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SMHandler Suite")
}

var _ = Describe("Handler", func() {
	var fakeHandler *webfakes.FakeHandler
	var handler *HTTPHandler
	var httpRecorder *httptest.ResponseRecorder

	BeforeEach(func() {
		httpRecorder = httptest.NewRecorder()
		handler = NewHTTPHandler(fakeHandler)
	})

	makeRequest := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request, _ := httptest.NewRequest(method, path, strings.NewReader(body))
		request, _ := http.NewRequest("GET", "/v2/catalog", nil)

		request.Header.Add("X-Broker-API-Version", "2.13")
		request.SetBasicAuth(credentials.Username, credentials.Password)
		request = request.WithContext(ctx)
		brokerAPI.ServeHTTP(recorder, request)
		return recorder
	}

	createFakeRequest := func(i, b string) *http.Request {
		body := bytes.NewBufferString("")
		uri := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", i, b)
		return httptest.NewRequest("GET", uri, body)
	}

	Describe("ServeHttp", func() {
		Context()
		Context("when http request is GET", func() {

		})

		Context("when http request is PUT/PATCH/POST and has invalid media type", func() {

		})

		Context("when http request is PUT/PATCH/POST and has invalid json body", func() {

		})

		Context("when web handler returns an error", func() {

		})
	})

	Describe("Handle", func() {
		It("invokes the underlying web.Handler", func() {

		})
	})

})
