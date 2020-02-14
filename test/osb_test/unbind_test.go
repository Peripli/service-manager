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
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	"net/http"

	. "github.com/onsi/ginkgo"
)

var _ = XDescribe("Unbind", func() {
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

	Context("platform_id check", func() {
		BeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

			brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/bid").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)
		})

		Context("unbind from not an instance owner", func() {
			var NewPlatformExpect *httpexpect.Expect

			BeforeEach(func() {
				platformJSON := common.MakePlatform("tcb-platform-test2", "tcb-platform-test2", "platform-type", "test-platform")
				platform := common.RegisterPlatformInSM(platformJSON, ctx.SMWithOAuth, map[string]string{})
				NewPlatformExpect = ctx.SM.Builder(func(req *httpexpect.Request) {
					username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
					req.WithBasicAuth(username, password)
				})
			})

			It("should return 404", func() {
				brokerServer.BindingHandler = parameterizedHandler(http.StatusOK, `{}`)
				NewPlatformExpect.DELETE(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/bid").
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusNotFound)
			})

			AfterEach(func() {
				ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/tcb-platform-test2").Expect().Status(http.StatusOK)
			})
		})
	})
})
