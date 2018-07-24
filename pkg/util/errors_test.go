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
	"errors"
	"net/http"

	"net/http/httptest"

	. "github.com/onsi/gomega"

	"encoding/json"

	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Errors", func() {

	var (
		responseRecorder *httptest.ResponseRecorder
		fakeErrorWriter  *errorResponseWriter
		testHTTPError    *HTTPError
	)

	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()
		fakeErrorWriter = &errorResponseWriter{}
		testHTTPError = &HTTPError{
			ErrorType:   "test error",
			Description: "test description",
			StatusCode:  http.StatusTeapot,
		}
	})

	Describe("HandleAPIError", func() {
		Context("when parameter is HTTPError", func() {
			It("writes to response writer the proper output", func() {
				HandleAPIError(testHTTPError, responseRecorder)

				Expect(responseRecorder.Code).To(Equal(http.StatusTeapot))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("test description"))
			})
		})
		Context("With error as parameter", func() {
			It("Writes to response writer the proper output", func() {
				HandleAPIError(errors.New("must not be included"), responseRecorder)

				Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Internal server error"))
				Expect(string(responseRecorder.Body.String())).ToNot(ContainSubstring("must not be included"))
			})
		})

		Context("With broken writer", func() {
			It("Logs write error", func() {
				hook := &loggingInterceptorHook{}
				logrus.AddHook(hook)
				HandleAPIError(errors.New(""), fakeErrorWriter)

				Expect(string(hook.data)).To(ContainSubstring("Could not write error to response: write error"))
			})
		})
	})

	Describe("HandleClientError", func() {
		Context("when response contains HTTPError", func() {
			It("returns an HTTPError containing the same error information", func() {
				bytes, err := json.Marshal(testHTTPError)
				Expect(err).ShouldNot(HaveOccurred())

				response := &http.Response{
					StatusCode: testHTTPError.StatusCode,
					Body:       closer(string(bytes)),
				}
				Expect(err).ShouldNot(HaveOccurred())

				err = HandleClientResponseError(response)
				validateHTTPErrorOccured(err, response.StatusCode)

			})
		})

		Context("when response contains standard error", func() {
			It("returns an error containing information about the error handling failure", func() {
				e := fmt.Errorf("test error")
				response := &http.Response{
					StatusCode: http.StatusTeapot,
					Body:       closer(e.Error()),
				}

				err := HandleClientResponseError(response)
				Expect(err.Error()).To(ContainSubstring("error handling client response error"))
			})
		})

		Describe("HTTPError", func() {
			var err *HTTPError
			BeforeEach(func() {
				err = &HTTPError{
					ErrorType:   "err",
					Description: "err",
					StatusCode:  http.StatusTeapot,
				}
			})

			It("implements the error interface by returning the description", func() {
				Expect(err.Error()).To(Equal(err.Description))
			})
		})
	})
})

type loggingInterceptorHook struct {
	data []byte
}

func (*loggingInterceptorHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *loggingInterceptorHook) Fire(entry *logrus.Entry) error {
	str, _ := entry.String()
	hook.data = append(hook.data, []byte(str)...)
	return nil
}

type errorResponseWriter struct {
}

func (errorResponseWriter) Header() http.Header {
	return http.Header{}
}

func (errorResponseWriter) Write([]byte) (int, error) {
	return -1, errors.New("write error")
}

func (errorResponseWriter) WriteHeader(statusCode int) {
	// do nothing
}
