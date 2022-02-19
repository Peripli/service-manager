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

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Post Binding Adapt Credentials", func() {
	Context("when call to working service broker", func() {
		It("should succeed", func() {
			brokerServer.BindingAdaptCredentialsHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.POST(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).WithJSON(&common.Object{}).
				Expect().Status(http.StatusOK)
		})
	})

	Context("when call to broken service broker", func() {
		It("should fail", func() {
			brokerServer.BindingAdaptCredentialsHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)
			assertFailingBrokerError(
				ctx.SMWithBasic.POST(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).WithJSON(&common.Object{}).
					Expect(), http.StatusInternalServerError, "internal server error")

		})
	})

	Context("when call to missing service broker", func() {
		It("should fail", func() {
			assertMissingBrokerError(
				ctx.SMWithBasic.POST("http://localhost:3456/v1/osb/123"+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).WithJSON(&common.Object{}).Expect())

		})
	})

	Context("when call to stopped service broker", func() {
		It("should fail", func() {
			assertUnresponsiveBrokerError(
				ctx.SMWithBasic.POST(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).WithJSON(&common.Object{}).Expect())
		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.BindingAdaptCredentialsHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.POST(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).WithQuery(headerKey, headerValue).Expect().Status(http.StatusOK)
		})
	})

	Context("when broker doesn't respond in a timely manner", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.BindingAdaptCredentialsHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.POST(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect())
		}, testTimeout)
	})
})
