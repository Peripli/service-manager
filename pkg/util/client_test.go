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
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Client Utils", func() {
	var (
		requestFunc  func(*http.Request, *http.Client) (*http.Response, error)
		reaction     *common.HTTPReaction
		expectations *common.HTTPExpectations
	)

	BeforeEach(func() {
		reaction = &common.HTTPReaction{}
		expectations = &common.HTTPExpectations{}
		requestFunc = common.DoHTTP(reaction, expectations)
	})

	Describe("SendRequest", func() {
		Context("when marshaling request body fails", func() {
			It("returns an error", func() {
				body := testTypeErrorMarshaling{
					Field: "Value",
				}
				_, err := util.SendRequest(context.TODO(), requestFunc, "GET", "http://example.com", map[string]string{}, body, http.DefaultClient)

				Expect(err).Should(HaveOccurred())
			})

		})

		Context("when method is invalid", func() {
			It("returns an error", func() {
				_, err := util.SendRequest(context.TODO(), requestFunc, "?+?.>", "http://example.com", map[string]string{}, nil, http.DefaultClient)

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
				expectations.Params = params
				expectations.Body = `{"field":"value"}`

				reaction.Err = nil
				reaction.Status = http.StatusOK

				resp, err := util.SendRequest(context.TODO(), requestFunc, "POST", "http://example.com", params, body, http.DefaultClient)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Context("When context has correlation id", func() {
			It("should attach it as header", func() {
				expectations.URL = "http://example.com"

				reaction.Err = nil
				reaction.Status = http.StatusOK

				expectedCorrelationID := "correlation-id"
				entry := logrus.NewEntry(logrus.StandardLogger())
				entry = entry.WithField(log.FieldCorrelationID, expectedCorrelationID)
				ctx := log.ContextWithLogger(context.TODO(), entry)
				resp, err := util.SendRequest(ctx, requestFunc, "GET", "http://example.com", nil, nil, http.DefaultClient)

				correlationID := resp.Request.Header.Get(log.CorrelationIDHeaders[0])
				Expect(correlationID).To(Equal(expectedCorrelationID))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Context("When sending a request with a header", func() {
		BeforeEach(func() {
			expectations.URL = "http://example.com"

			reaction.Err = nil
			reaction.Status = http.StatusOK
		})

		It("should attach it as header", func() {
			ctx := context.TODO()
			resp, err := util.SendRequestWithHeaders(ctx, requestFunc, "GET", "http://example.com", nil, nil, map[string]string{
				"header": "header",
			}, http.DefaultClient)

			header := resp.Request.Header.Get("header")
			Expect(header).To(Equal("header"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Describe("BodyToObject", func() {
		var resp *http.Response
		var err error

		BeforeEach(func() {
			reaction.Err = nil
			reaction.Status = http.StatusOK
			reaction.Body = `{"field":"value"}`

			resp, err = util.SendRequest(context.TODO(), requestFunc, "POST", "http://example.com", map[string]string{}, nil, http.DefaultClient)
			Expect(err).ShouldNot(HaveOccurred())
		})

		Context("when unmarshaling fails", func() {
			It("returns an error", func() {
				var val testType
				err = util.BodyToObject(resp.Body, val)

				Expect(err).Should(HaveOccurred())
			})
		})

		It("reads the client response content", func() {
			var val testType
			err = util.BodyToObject(resp.Body, &val)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(val.Field).To(Equal("value"))
		})
	})
})

type testTypeErrorMarshaling struct {
	Field string `json:"field"`
}

func (testTypeErrorMarshaling) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("error")
}

type testType struct {
	Field string `json:"field"`
}
