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

package rest

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
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
	return -1, errors.New("write error")
}

func (errorResponseWriter) WriteHeader(statusCode int) {
	// do nothing
}

var _ = Describe("Errors", func() {

	mockedWriter := &mockedResponseWriter{}
	mockedErrorWriter := &errorResponseWriter{}

	BeforeEach(func() {
		mockedWriter.data = []byte{}
	})

	Describe("HandleError", func() {
		Context("With ErrorResponse as parameter", func() {
			It("Writes to response writer the proper output", func() {
				HandleError(&types.ErrorResponse{ErrorType: "test error", Description: "test description", StatusCode: http.StatusAccepted}, mockedWriter)
				Expect(string(mockedWriter.data)).To(ContainSubstring("test description"))
				Expect(mockedWriter.status).To(Equal(http.StatusAccepted))
			})
		})
		Context("With error as parameter", func() {
			It("Writes to response writer the proper output", func() {
				HandleError(errors.New("must not be included"), mockedWriter)
				Expect(string(mockedWriter.data)).To(ContainSubstring("Internal server error"))
				Expect(string(mockedWriter.data)).ToNot(ContainSubstring("must not be included"))
				Expect(mockedWriter.status).To(Equal(http.StatusInternalServerError))
			})
		})
		Context("With broken writer", func() {
			It("Logs write error", func() {
				hook := &loggingInterceptorHook{}
				logrus.AddHook(hook)
				HandleError(errors.New(""), mockedErrorWriter)
				Expect(string(hook.data)).To(ContainSubstring("Could not write error to response: write error"))
			})
		})
	})
})
