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

package rest_test

import (
	"errors"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/rest"
)

func TestRest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rest Suite")
}

type mockedResponseWriter struct {
	data   []byte
	status int
}

func (writer *mockedResponseWriter) Header() http.Header {
	return http.Header{}
}

func (writer *mockedResponseWriter) Write(data []byte) (int, error) {
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *mockedResponseWriter) WriteHeader(status int) {
	writer.status = status
}

var _ = Describe("Errors", func() {

	mockedWriter := &mockedResponseWriter{}

	BeforeEach(func() {
		mockedWriter.data = []byte{}
	})

	Describe("Send JSON", func() {
		Context("With valid parameters", func() {
			It("Writes to response writer", func() {
				response := rest.ErrorResponse{Error: "test error", Description: "test description"}
				if err := rest.SendJSON(mockedWriter, 200, response); err != nil {
					Fail("Serializing valid ErrorResponse should be successful")
				}
				Expect(string(mockedWriter.data)).To(ContainSubstring("test description"))
			})
		})
	})

	Describe("Handle Error", func() {
		Context("With nil error", func() {
			It("Should have no data in Response Writer", func() {
				rest.HandlerError(nil, mockedWriter)
				Expect(string(mockedWriter.data)).To(BeEmpty())
			})
		})
		Context("With an error", func() {
			It("Should write to Response Writer", func() {
				rest.HandlerError(errors.New("test description"), mockedWriter)
				Expect(string(mockedWriter.data)).To(ContainSubstring("test description"))
				Expect(mockedWriter.status).To(Equal(http.StatusInternalServerError))
			})
		})
	})

	Describe("Error Handler Func", func() {

		var returnedData error

		var mockedAPIHandler rest.APIHandler

		BeforeEach(func() {
			returnedData = nil
		})

		JustBeforeEach(func() {
			mockedAPIHandler = func(writer http.ResponseWriter, request *http.Request) error {
				return returnedData
			}
		})

		Context("With API Handler not returning an error", func() {
			It("Should have no data in Response Writer", func() {
				httpHandler := rest.ErrorHandlerFunc(mockedAPIHandler)
				httpHandler.ServeHTTP(mockedWriter, nil)
				Expect(string(mockedWriter.data)).To(BeEmpty())
			})
		})

		Context("With API Handler returning an error", func() {
			BeforeEach(func() {
				returnedData = errors.New("test description")
			})

			It("Should write to Response Writer", func() {
				httpHandler := rest.ErrorHandlerFunc(mockedAPIHandler)
				httpHandler.ServeHTTP(mockedWriter, nil)
				Expect(string(mockedWriter.data)).To(ContainSubstring("test description"))
				Expect(mockedWriter.status).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
