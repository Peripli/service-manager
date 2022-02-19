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
	"bytes"
	"net/http/httptest"
	"strconv"

	"strings"

	"net/http"

	"encoding/json"

	"fmt"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	bodyMaxSize = 2000000
)

func generateJSON(size int) string {
	var result bytes.Buffer
	result.WriteString(`{`)
	size = size / 16
	for i := 0; i < size-1; i++ {
		result.WriteString(fmt.Sprintf(`"property%s":"value%s",`, strconv.Itoa(int(size)), strconv.Itoa(int(size))))
	}
	result.WriteString(fmt.Sprintf(`"property%s":"value%s"`, strconv.Itoa(int(size)), strconv.Itoa(int(size))))
	result.WriteString(`}`)

	return result.String()
}

var _ = Describe("Handler", func() {
	const validJSON = `{"key1":"value1","key2":"value2"}`
	const invalidJSON = `{{{"KEY"`

	var fakeHandler *webfakes.FakeHandler
	var handler *HTTPHandler
	var responseRecorder *httptest.ResponseRecorder

	BeforeEach(func() {
		fakeHandler = &webfakes.FakeHandler{}
		responseRecorder = httptest.NewRecorder()
		handler = NewHTTPHandler(fakeHandler, bodyMaxSize)
	})

	makeRequest := func(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))

		for k, v := range headers {
			request.Header.Add(k, v)
		}
		handler.ServeHTTP(responseRecorder, request)
		return responseRecorder
	}

	validateHTTPErrorOccurred := func(response *httptest.ResponseRecorder, expectedStatusCode int) {
		Expect(response.Code).To(Equal(expectedStatusCode))

		var body util.HTTPError
		decoder := json.NewDecoder(response.Body)
		err := decoder.Decode(&body)
		Expect(err).ToNot(HaveOccurred())
		Expect(body.ErrorType).To(Not(BeEmpty()))
		Expect(body.Description).To(Not(BeEmpty()))
	}

	Describe("ServeHTTP", func() {
		Context("when http request has invalid media type", func() {
			Specify("response contains a proper HTTPError", func() {
				response := makeRequest(http.MethodPost, "http://example.com", validJSON, map[string]string{
					"Content-Type": "application/xml",
				})

				validateHTTPErrorOccurred(response, http.StatusUnsupportedMediaType)
			})
		})

		Context("when http request has invalid json body", func() {
			Specify("response contains a proper HTTPError", func() {
				response := makeRequest(http.MethodPost, "http://example.com", invalidJSON, map[string]string{
					"Content-Type": "application/json",
				})

				validateHTTPErrorOccurred(response, http.StatusBadRequest)
			})
		})

		Context("when http request has too large body", func() {
			Specify("response contains a proper HTTPError", func() {
				var bodySize int = 2100000
				response := makeRequest(http.MethodPost, "http://example.com", generateJSON(bodySize), map[string]string{
					"Content-Type": "application/json",
				})

				validateHTTPErrorOccurred(response, http.StatusRequestEntityTooLarge)
			})
		})

		Context("when call to web handler returns an error", func() {
			Specify("response contains a proper HTTPError", func() {
				handlerError := fmt.Errorf("error")
				fakeHandler.HandleReturns(nil, handlerError)

				response := makeRequest(http.MethodPost, "http://example.com", validJSON, map[string]string{
					"Content-Type": "application/json",
				})

				validateHTTPErrorOccurred(response, http.StatusInternalServerError)
			})
		})

		Context("when call to web handler is successful", func() {
			var fakeHandlerResponse *web.Response

			BeforeEach(func() {
				fakeHandlerResponse = &web.Response{
					StatusCode: http.StatusOK,
				}

				fakeHandler.HandleReturns(fakeHandlerResponse, nil)
			})

			Context("when headers are present ин the web.Handler's response", func() {
				BeforeEach(func() {
					headers := http.Header{}
					headers.Add("Content-Length", "52")
					headers.Add("Random-Header", "random-value")
					fakeHandlerResponse.Header = headers

					fakeHandler.HandleReturns(fakeHandlerResponse, nil)

				})

				Specify("Content-Length header is not copied to the HTTPHandler's response", func() {
					response := makeRequest("", "http://example.com", "", map[string]string{})

					header := response.Header().Get("Content-Length")
					Expect(header).To(BeEmpty())
				})

				Specify("other headers are copied to the HTTPHandler's response", func() {
					response := makeRequest("", "http://example.com", "", map[string]string{})

					Expect(response.Header().Get("Random-Header")).ToNot(BeEmpty())
				})
			})

			It("propagates the response from the web.Handler to the HTTPHandler", func() {
				response := makeRequest("", "http://example.com", "", map[string]string{})

				Expect(response.Code).To(Equal(fakeHandlerResponse.StatusCode))
			})
		})
	})

	Describe("Handle", func() {
		var (
			expectedResponse *web.Response
			request          *web.Request
		)

		BeforeEach(func() {
			request = &web.Request{
				Request:    httptest.NewRequest("", "http://example.com", strings.NewReader("")),
				PathParams: map[string]string{},
				Body:       []byte("{}"),
			}

			expectedResponse = &web.Response{
				StatusCode: http.StatusOK,
			}

			fakeHandler.HandleReturns(expectedResponse, nil)
		})

		It("invokes the underlying web.Handler", func() {
			resp, err := handler.Handle(request)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp).To(Equal(expectedResponse))
			Expect(fakeHandler.HandleCallCount()).To(Equal(1))
		})
	})

})
