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

var _ = Describe("Get Binding Last Operation", func() {
	Context("when call to working service broker", func() {
		It("should succeed", func() {
			brokerServer.BindingLastOpHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)
		})
	})

	Context("when call to failing service broker", func() {
		It("should fail", func() {
			brokerServer.BindingLastOpHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)
			assertFailingBrokerError(
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect(), http.StatusInternalServerError, "internal server error")

		})
	})

	Context("when call to missing service broker", func() {
		It("should fail", func() {
			assertMissingBrokerError(
				ctx.SMWithBasic.GET("http://localhost:3456/v1/osb/123"+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).Expect())
		})
	})

	Context("when call to stopped service broker", func() {
		It("should fail", func() {
			assertUnresponsiveBrokerError(
				ctx.SMWithBasic.GET(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).Expect())
		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.BindingHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).WithQuery(headerKey, headerValue).Expect().Status(http.StatusOK)
		})
	})

	Context("when broker doesn't respond in a timely manner", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.BindingLastOpHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect())
		})
	})
})
