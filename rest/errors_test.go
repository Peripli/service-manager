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

package rest_test

import (
	"errors"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

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
	return -1, errors.New("err")
}

func (errorResponseWriter) WriteHeader(statusCode int) {
	// do nothing
}

var _ = Describe("Errors", func() {

	mockedWriter := &mockedResponseWriter{}
	testError := errors.New("test description")

	BeforeEach(func() {
		mockedWriter.data = []byte{}
	})

	Describe("Send JSON", func() {
		Context("With valid parameters", func() {
			It("Writes to response writer", func() {
				response := rest.ErrorResponse{Error: "test error", Description: "test description"}
				if err := rest.SendJSON(mockedWriter, http.StatusOK, response); err != nil {
					Fail("Serializing valid ErrorResponse should be successful")
				}
				Expect(string(mockedWriter.data)).To(ContainSubstring(testError.Error()))
			})
		})
	})

	Describe("Handle Error", func() {
		Context("With nil error", func() {
			It("Should have no data in Response Writer", func() {
				rest.HandleError(nil, mockedWriter)
				Expect(string(mockedWriter.data)).To(BeEmpty())
			})
		})

		Context("With an error", func() {
			It("Should write to Response Writer", func() {
				rest.HandleError(testError, mockedWriter)
				Expect(string(mockedWriter.data)).To(ContainSubstring(testError.Error()))
				Expect(mockedWriter.status).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("With SendJSON returning an error", func() {
			It("Should not write to Response Writer and log", func() {
				hook := &loggingInterceptorHook{}
				logrus.AddHook(hook)
				rest.HandleError(testError, &errorResponseWriter{})
				Expect(hook.data).To(ContainSubstring("Could not write error to response"))
			})
		})
	})

})
