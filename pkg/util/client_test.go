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

package util

import (
	"bytes"
	"net/http"

	"io"

	"io/ioutil"

	"fmt"

	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client Utils", func() {
	doHTTP := func(reaction *httpReaction, checks *httpExpectations) func(*http.Request) (*http.Response, error) {
		return func(request *http.Request) (*http.Response, error) {

			if len(checks.URL) > 0 && !strings.Contains(checks.URL, request.URL.Host) {
				Fail(fmt.Sprintf("unexpected URL; expected %v, got %v", checks.URL, request.URL.Path))
			}

			for k, v := range checks.headers {
				actualValue := request.Header.Get(k)
				if e, a := v, actualValue; e != a {
					Fail(fmt.Sprintf("unexpected header value for key %q; expected %v, got %v", k, e, a))
				}
			}

			for k, v := range checks.params {
				actualValue := request.URL.Query().Get(k)
				if e, a := v, actualValue; e != a {
					Fail(fmt.Sprintf("unexpected parameter value for key %q; expected %v, got %v", k, e, a))
				}
			}

			var bodyBytes []byte
			if request.Body != nil {
				var err error
				bodyBytes, err = ioutil.ReadAll(request.Body)
				if err != nil {
					Fail(fmt.Sprintf("error reading request body bytes: %v", err))
				}
			}

			if e, a := checks.body, string(bodyBytes); e != a {
				Fail(fmt.Sprintf("unexpected request body: expected %v, got %v", e, a))
			}

			return &http.Response{
				StatusCode: reaction.status,
				Body:       closer(reaction.body),
			}, reaction.err
		}
	}

	var (
		requestFunc  doRequestFunc
		reaction     *httpReaction
		expectations *httpExpectations
	)

	BeforeEach(func() {
		reaction = &httpReaction{}
		expectations = &httpExpectations{}
		requestFunc = doHTTP(reaction, expectations)
	})

	Describe("SendRequest", func() {
		Context("when marshaling request body fails", func() {
			It("returns an error", func() {
				body := testTypeErrorMarshaling{
					Field: "Value",
				}
				_, err := SendRequest(requestFunc, "GET", "http://example.com", map[string]string{}, body)

				Expect(err).Should(HaveOccurred())
			})

		})

		Context("when method is invalid", func() {
			It("returns an error", func() {
				_, err := SendRequest(requestFunc, "?+?.>", "http://example.com", map[string]string{}, nil)

				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when request that has parameters and body is successful", func() {
			It("returns no error", func() {
				params := map[string]string{
					"key": "val",
				}
				body := struct {
					Field string `json:"field"`
				}{Field: "value"}

				expectations.URL = "http://example.com"
				expectations.params = params
				expectations.body = `{"field":"value"}`

				reaction.err = nil
				reaction.status = http.StatusOK

				resp, err := SendRequest(requestFunc, "POST", "http://example.com", params, body)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("BodyToObject", func() {
		var resp *http.Response
		var err error

		BeforeEach(func() {
			reaction.err = nil
			reaction.status = http.StatusOK
			reaction.body = `{"field":"value"}`

			resp, err = SendRequest(requestFunc, "POST", "http://example.com", map[string]string{}, nil)
			Expect(err).ShouldNot(HaveOccurred())
		})

		Context("when unmarshaling fails", func() {
			It("returns an error", func() {
				var val testType
				err = BodyToObject(val, resp.Body)

				Expect(err).Should(HaveOccurred())
			})
		})

		It("reads the client response content", func() {
			var val testType
			err = BodyToObject(&val, resp.Body)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(val.Field).To(Equal("value"))
		})
	})
})

type httpReaction struct {
	status int
	body   string
	err    error
}

type httpExpectations struct {
	URL     string
	body    string
	params  map[string]string
	headers map[string]string
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func closer(s string) io.ReadCloser {
	return nopCloser{bytes.NewBufferString(s)}
}

type testTypeErrorMarshaling struct {
	Field string `json:"field"`
}

func (testTypeErrorMarshaling) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("error")
}

type testType struct {
	Field string `json:"field"`
}
