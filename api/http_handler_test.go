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
	"testing"

	"net/http/httptest"

	"strings"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTPHandler Suite")
}

var _ = Describe("Handler", func() {
	const validJSON = `{"key1":"value1","key2":"value2"}`
	const invalidJSON = `{{{"KEY"`

	var fakeHandler *webfakes.FakeHandler
	var fakeHandlerWebResponse *web.Response
	var fakeHandlerError error
	var handler *HTTPHandler
	var actualResponse *httptest.ResponseRecorder

	BeforeEach(func() {
		actualResponse = httptest.NewRecorder()
		fakeHandler.HandleReturns(fakeHandlerWebResponse, fakeHandlerError)

		handler = NewHTTPHandler(fakeHandler)
	})

	makeRequest := func(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, strings.NewReader(body))

		for k, v := range headers {
			request.Header.Add(k, v)
		}
		handler.ServeHTTP(recorder, request)
		return recorder
	}

	Describe("ServeHTTP", func() {
		Context("when http request has invalid media type", func() {
			Specify("actualResponse contains a proper HTTPError", func() {

			})
		})

		Context("when http request has invalid json body", func() {
			Specify("actualResponse contains a proper HTTPError", func() {

			})
		})

		Context("when call to web handler returns an error", func() {
			Specify("actualResponse contains a proper HTTPError", func() {

			})
		})

		Context("when call to web handler is successful", func() {
			Context("when writing to ResponseWriter fails", func() {
				Specify("actualResponse contains a proper HTTPError", func() {

				})
			})

			Context("when headers are present the web.Handler's actualResponse", func() {
				Specify("Content-Length header is not copied to the HTTPHandler's actualResponse", func() {

				})

				Specify("other headers are copied to the HTTPHandler's actualResponse", func() {

				})
			})

			It("propagates the actualResponse from the web.Handler to the HTTPHandler", func() {

			})
		})
	})

	Describe("Handle", func() {
		It("invokes the underlying web.Handler", func() {
			handler.Handle(&web.Request{
				Request:    httptest.NewRequest("", "", strings.NewReader("")),
				PathParams: map[string]string{},
				Body:       []byte("{}"),
			})

			Expect(fakeHandler.HandleCallCount()).To(Equal(1))
		})
	})

})
