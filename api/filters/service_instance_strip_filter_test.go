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

package filters

import (
	"net/http"

	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service Instance Strip Filter", func() {
	const (
		propertyNotToBeDeleted = "some_prop"
		defaultValue           = "value"
		invalidJSON            = "invalid json"
	)
	var (
		filter                    ServiceInstanceStripFilter
		handler                   *webfakes.FakeHandler
		jsonWithPropertiesToStrip string
	)

	BeforeEach(func() {
		filter = ServiceInstanceStripFilter{}
		handler = &webfakes.FakeHandler{}
		jsonWithPropertiesToStrip = `{}`
	})

	Context("Create instance", func() {
		When("body has properties which cannot be set", func() {
			It("should remove them from request body as json", func() {
				var err error
				for _, prop := range serviceInstanceUnmodifiableProperties {
					jsonWithPropertiesToStrip, err = sjson.Set(jsonWithPropertiesToStrip, prop, defaultValue)
					Expect(err).ToNot(HaveOccurred())
				}
				jsonWithPropertiesToStrip, err = sjson.Set(jsonWithPropertiesToStrip, propertyNotToBeDeleted, defaultValue)
				Expect(err).ToNot(HaveOccurred())

				req := mockedRequest(http.MethodPost, jsonWithPropertiesToStrip)
				_, err = filter.Run(req, handler)
				Expect(err).ToNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
				requestBody := handler.HandleArgsForCall(0).Body
				for _, prop := range serviceInstanceUnmodifiableProperties {
					Expect(gjson.GetBytes(requestBody, prop).String()).To(BeEmpty())
				}
				Expect(gjson.GetBytes(requestBody, propertyNotToBeDeleted).String()).To(Equal(defaultValue))
			})

			It("should remove them from request body as string with duplicate properties", func() {
				var err error
				stringBody := "{\"ready\":\"value\",\"ready\":\"value2\",\"usable\":\"value\",\"context\":\"value\",\"some_prop\":\"value\"}"
				req := mockedRequest(http.MethodPost, stringBody)
				_, err = filter.Run(req, handler)
				Expect(err).ToNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
				requestBody := handler.HandleArgsForCall(0).Body
				for _, prop := range serviceInstanceUnmodifiableProperties {
					Expect(gjson.GetBytes(requestBody, prop).String()).To(BeEmpty())
				}
				Expect(gjson.GetBytes(requestBody, propertyNotToBeDeleted).String()).To(Equal(defaultValue))
			})
		})
		When("body is invalid json", func() {
			It("should do nothing", func() {
				req := mockedRequest(http.MethodPost, invalidJSON)
				filter.Run(req, handler)
				Expect(handler.HandleCallCount()).To(Equal(1))
				requestBody := handler.HandleArgsForCall(0).Body
				Expect(string(requestBody)).To(Equal(invalidJSON))
			})
		})
	})

	Context("Create instance", func() {
		When("body has properties which cannot be set", func() {
			It("should remove them from request body", func() {
				var err error
				for _, prop := range serviceInstanceUnmodifiableProperties {
					jsonWithPropertiesToStrip, err = sjson.Set(jsonWithPropertiesToStrip, prop, defaultValue)
					Expect(err).ToNot(HaveOccurred())
				}
				jsonWithPropertiesToStrip, err = sjson.Set(jsonWithPropertiesToStrip, propertyNotToBeDeleted, defaultValue)
				Expect(err).ToNot(HaveOccurred())

				req := mockedRequest(http.MethodPatch, jsonWithPropertiesToStrip)
				_, err = filter.Run(req, handler)
				Expect(err).ToNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
				requestBody := handler.HandleArgsForCall(0).Body
				for _, prop := range serviceInstanceUnmodifiableProperties {
					Expect(gjson.GetBytes(requestBody, prop).String()).To(BeEmpty())
				}
				Expect(gjson.GetBytes(requestBody, propertyNotToBeDeleted).String()).To(Equal(defaultValue))
			})
		})
		When("body is invalid json", func() {
			It("should do nothing", func() {
				req := mockedRequest(http.MethodPatch, invalidJSON)
				filter.Run(req, handler)
				Expect(handler.HandleCallCount()).To(Equal(1))
				requestBody := handler.HandleArgsForCall(0).Body
				Expect(string(requestBody)).To(Equal(invalidJSON))
			})
		})
	})
})
