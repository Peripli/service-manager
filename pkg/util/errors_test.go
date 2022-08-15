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

package util_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util/utilfakes"

	"net/http/httptest"

	. "github.com/onsi/gomega"

	"encoding/json"

	. "github.com/onsi/ginkgo"
	"github.com/sirupsen/logrus"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/test/common"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/test/testutil"
)

//go:generate counterfeiter io.ReadCloser

var _ = Describe("Errors", func() {

	var (
		responseRecorder *httptest.ResponseRecorder
		fakeErrorWriter  *errorResponseWriter
		testHTTPError    *util.HTTPError
		ctx              context.Context
	)

	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()
		fakeErrorWriter = &errorResponseWriter{}
		testHTTPError = &util.HTTPError{
			ErrorType:   "test error",
			Description: "test description",
			StatusCode:  http.StatusTeapot,
		}
		ctx = context.TODO()
	})

	Describe("WriteError", func() {
		Context("when parameter is HTTPError", func() {
			It("writes to response writer the proper output", func() {
				util.WriteError(ctx, testHTTPError, responseRecorder)

				Expect(responseRecorder.Code).To(Equal(http.StatusTeapot))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("test description"))
			})
		})
		Context("With error as parameter", func() {
			It("Writes to response writer the proper output", func() {
				util.WriteError(ctx, errors.New("must not be included"), responseRecorder)

				Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
				Expect(responseRecorder.Body.String()).To(ContainSubstring("Internal server error"))
				Expect(string(responseRecorder.Body.String())).ToNot(ContainSubstring("must not be included"))
			})
		})

		Context("With broken writer", func() {
			It("Logs write error", func() {
				hook := &testutil.LogInterceptor{}
				logrus.AddHook(hook)
				util.WriteError(ctx, errors.New(""), fakeErrorWriter)

				Expect(hook).To(ContainSubstring("Could not write error to response: write error"))
			})
		})
	})

	Describe("HandleResponseError", func() {
		Context("When response body cannot be read", func() {
			It("return an error with the error in the body", func() {
				fakeBody := &utilfakes.FakeReadCloser{}
				readError := errors.New("read-error")
				fakeBody.ReadReturns(0, readError)

				response := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       fakeBody,
				}
				err := util.HandleResponseError(response)
				Expect(err.Error()).To(SatisfyAll(
					ContainSubstring("500"), ContainSubstring(readError.Error())))
			})
		})

		Context("when response contains HTTPError", func() {
			It("returns an HTTPError containing the same error information", func() {
				bytes, err := json.Marshal(testHTTPError)
				Expect(err).ShouldNot(HaveOccurred())

				response := &http.Response{
					StatusCode: testHTTPError.StatusCode,
					Body:       common.Closer(string(bytes)),
				}
				Expect(err).ShouldNot(HaveOccurred())

				err = util.HandleResponseError(response)
				validateHTTPErrorOccurred(err, response.StatusCode)
			})
		})

		Context("when response contains standard error", func() {
			It("returns an error containing the response body and status code", func() {
				e := "test error"
				response := &http.Response{
					StatusCode: http.StatusTeapot,
					Body:       common.Closer(e),
				}

				err := util.HandleResponseError(response)
				Expect(err.Error()).To(ContainSubstring(e))
			})
		})

		Context("when response is linked to request", func() {
			It("returns an error containing information about the failing request", func() {
				e := "test error"
				u, _ := url.Parse("http://host/path")
				r := http.Request{
					Method: http.MethodPost,
					URL:    u,
				}
				response := &http.Response{
					StatusCode: http.StatusTeapot,
					Body:       common.Closer(e),
					Request:    &r,
				}

				err := util.HandleResponseError(response)
				Expect(err.Error()).To(SatisfyAll(
					ContainSubstring(r.Method),
					ContainSubstring(r.URL.String()),
					ContainSubstring(e)))
			})
		})

		Context("when response contains JSON error that has no description", func() {
			It("returns an error containing the response body", func() {
				e := `{"key":"value"}`
				response := &http.Response{
					StatusCode: http.StatusTeapot,
					Body:       common.Closer(e),
				}

				err := util.HandleResponseError(response)
				Expect(err.Error()).To(ContainSubstring(e))
			})
		})

		Describe("HandleStorageError", func() {
			Context("with no errors", func() {
				It("returns nil", func() {
					err := util.HandleStorageError(nil, "")

					Expect(err).To(Not(HaveOccurred()))
				})
			})

			Context("with unique constraint violation storage error", func() {
				It("returns proper HTTPError", func() {
					err := util.HandleStorageError(util.ErrAlreadyExistsInStorage, "entityName")

					validateHTTPErrorOccurred(err, http.StatusConflict)
				})
			})

			Context("with not found in storage error", func() {
				It("returns proper HTTPError", func() {
					err := util.HandleStorageError(util.ErrNotFoundInStorage, "entityName")

					validateHTTPErrorOccurred(err, http.StatusNotFound)
				})
			})

			Context("with unrecongized error", func() {
				It("propagates it", func() {
					e := errors.New("test error")
					err := util.HandleStorageError(e, "entityName")

					Expect(err.Error()).To(ContainSubstring(e.Error()))
				})
			})
		})

		Describe("HTTPError", func() {
			var err *util.HTTPError
			BeforeEach(func() {
				err = &util.HTTPError{
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
