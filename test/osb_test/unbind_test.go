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

package osb_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Unbind", func() {
	Context("when trying to delete binding", func() {
		It("should be successful", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQueryObject(provisionRequestBodyMap()()).
				Expect().Status(http.StatusOK).JSON().Object()

		})
	})

	Context("when call to failing service broker", func() {
		It("should return error", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)
			assertFailingBrokerError(
				ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMap()()).Expect(), http.StatusInternalServerError, `internal server error`)
		})
	})

	Context("when call to missing broker", func() {
		It("unbind fails", func() {
			assertMissingBrokerError(
				ctx.SMWithBasic.DELETE("http://localhost:3456/v1/osb/123"+"/v2/service_instances/iid/service_bindings/bid").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithQueryObject(provisionRequestBodyMap()()).Expect())
		})
	})

	Context("when call to stopped service broker", func() {
		It("should fail", func() {
			assertUnresponsiveBrokerError(
				ctx.SMWithBasic.DELETE(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithQueryObject(provisionRequestBodyMap()()).Expect())

		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.BindingHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).WithQuery(headerKey, headerValue).Expect().Status(http.StatusOK)
		})
	})

	Context("when broker doesn't respond in a timely manner", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.BindingHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQueryObject(provisionRequestBodyMap()()).
				Expect())
		})
	})
})
