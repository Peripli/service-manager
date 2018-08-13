/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package proxy_test

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/Peripli/service-manager/pkg/proxy"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Proxy Suite")
}

var _ = Describe("proxy", func() {

	var statusCode int
	var server *ghttp.Server
	var proxier *proxy.Proxy
	var response []byte

	BeforeSuite(func() {
		statusCode = 200
		response = []byte("OK")
		server = fakeServer(&statusCode, &response)
	})

	BeforeEach(func() {
		proxier = proxy.NewReverseProxy(proxy.Options{
			Transport: http.DefaultTransport,
		})
	})

	It("GET method without body", func() {
		req, reqBuilder := buildProxyRequest("GET", server.URL(), proxier)

		checkProxyRequest(proxier, req, reqBuilder, nil, response)
		common.VerifyReqReceived(server, 1, http.MethodGet, "/")
	})

	It("POST method with body", func() {
		req, reqBuilder := buildProxyRequest("POST", server.URL(), proxier)
		body := []byte("DATA")
		response = body

		checkProxyRequest(proxier, req, reqBuilder, body, response)
		common.VerifyReqReceived(server, 1, http.MethodPost, "/")
	})

})

func buildProxyRequest(method, rawURL string, proxier *proxy.Proxy) (*http.Request, *proxy.RequestBuilder) {
	req, err := http.NewRequest(method, "http://test.com", nil)
	Expect(err).ShouldNot(HaveOccurred())

	urlObject, err := url.Parse(rawURL)
	Expect(err).ShouldNot(HaveOccurred())
	return req, proxier.RequestBuilder().URL(urlObject)
}

func checkProxyRequest(
	proxier *proxy.Proxy,
	req *http.Request,
	reqBuilder *proxy.RequestBuilder,
	requestBody []byte,
	responseBody []byte) {
	resp, err := proxier.ProxyRequest(req, reqBuilder, requestBody)

	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(200))
	respBytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(respBytes).To(Equal(responseBody))
}

func fakeServer(code *int, response interface{}) *ghttp.Server {
	server := ghttp.NewServer()
	server.RouteToHandler(http.MethodPost, regexp.MustCompile(".*"), ghttp.RespondWithPtr(code, response))
	server.RouteToHandler(http.MethodGet, regexp.MustCompile(".*"), ghttp.RespondWithPtr(code, response))

	return server
}
