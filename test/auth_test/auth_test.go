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

package auth_test

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuthentication(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authentication Tests Suite")
}

func expectUnauthorizedRequest(ctx *common.TestContext, method, path, authHeader string) {
	ctx.SM.Request(method, path).
		WithHeader("Authorization", authHeader).
		WithHeader("Content-type", "application/json").
		WithJSON(common.Object{}).
		Expect().
		Status(http.StatusUnauthorized).
		JSON().Object().Keys().Contains("error", "description")
}

var _ = Describe("Service Manager Authentication", func() {
	var (
		ctx *common.TestContext
	)

	BeforeSuite(func() {
		ctx = common.DefaultTestContext()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	Context("Nontrivial scenarios", func() {
		It("Forbidden broker registration with basic auth", func() {
			ctx.SMWithBasic.GET(web.ServiceBrokersURL).
				Expect().
				Status(http.StatusOK)

			ctx.SMWithBasic.POST(web.ServiceBrokersURL).
				WithHeader("Content-type", "application/json").
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusUnauthorized).
				JSON().Object().Keys().Contains("error", "description")
		})
	})

	Context("Failing security scenarios", func() {
		emptyAuthHeader := func() string {
			return ""
		}
		emptyBearerAuthHeader := func() string {
			return "Bearer "
		}
		invalidBasicAuthHeader := func() string {
			return "Basic abc"
		}
		invalidBearerAuthHeader := func() string {
			return "Bearer abc"
		}
		validBasicAuthHeader := func() string {
			auth := ctx.TestPlatform.Credentials.Basic.Username + ":" + ctx.TestPlatform.Credentials.Basic.Password
			return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		}
		authRequests := []struct {
			name, method, path string
			authHeader         func() string
		}{
			// PLATFORMS
			{"Missing authorization header", "GET", web.PlatformsURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.PlatformsURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.PlatformsURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.PlatformsURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.PlatformsURL + "/999/operations/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.PlatformsURL + "/999/operations/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.PlatformsURL + "/999/operations/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.PlatformsURL + "/999/operations/999", invalidBearerAuthHeader},
			{"Valid token in authorization header", "GET", web.PlatformsURL + "/999/operations/999", validBasicAuthHeader},

			{"Missing authorization header", "GET", web.PlatformsURL, emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.PlatformsURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.PlatformsURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.PlatformsURL, invalidBearerAuthHeader},

			{"Missing authorization header", "POST", web.PlatformsURL, emptyAuthHeader},
			{"Invalid authorization schema", "POST", web.PlatformsURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "POST", web.PlatformsURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "POST", web.PlatformsURL, invalidBearerAuthHeader},

			{"Missing authorization header", "PATCH", web.PlatformsURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "PATCH", web.PlatformsURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "PATCH", web.PlatformsURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PATCH", web.PlatformsURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "DELETE", web.PlatformsURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "DELETE", web.PlatformsURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "DELETE", web.PlatformsURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "DELETE", web.PlatformsURL + "/999", invalidBearerAuthHeader},

			// BROKERS
			{"Missing authorization header", "GET", web.ServiceBrokersURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceBrokersURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceBrokersURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceBrokersURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.ServiceBrokersURL + "/999/operations/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceBrokersURL + "/999/operations/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceBrokersURL + "/999/operations/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceBrokersURL + "/999/operations/999", invalidBearerAuthHeader},
			{"Valid token in authorization header", "GET", web.ServiceBrokersURL + "/999/operations/999", validBasicAuthHeader},

			{"Missing authorization header", "GET", web.ServiceBrokersURL, emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceBrokersURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceBrokersURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceBrokersURL, invalidBearerAuthHeader},

			{"Missing authorization header", "POST", web.ServiceBrokersURL, emptyAuthHeader},
			{"Invalid authorization schema", "POST", web.ServiceBrokersURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "POST", web.ServiceBrokersURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "POST", web.ServiceBrokersURL, invalidBearerAuthHeader},

			{"Missing authorization header", "PATCH", web.ServiceBrokersURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "PATCH", web.ServiceBrokersURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "PATCH", web.ServiceBrokersURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PATCH", web.ServiceBrokersURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "DELETE", web.ServiceBrokersURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "DELETE", web.ServiceBrokersURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "DELETE", web.ServiceBrokersURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "DELETE", web.ServiceBrokersURL + "/999", invalidBearerAuthHeader},

			// OSB
			{"Missing authorization header", "GET", "/v1/osb/999/v2/catalog", emptyAuthHeader},
			{"Invalid authorization schema", "GET", "/v1/osb/999/v2/catalog", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", "/v1/osb/999/v2/catalog", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", "/v1/osb/999/v2/catalog", invalidBearerAuthHeader},

			{"Missing authorization header", "PUT", "/v1/osb/999/v2/service_instances/111", emptyAuthHeader},
			{"Invalid authorization schema", "PUT", "/v1/osb/999/v2/service_instances/111", invalidBasicAuthHeader},
			{"Missing token in authorization header", "PUT", "/v1/osb/999/v2/service_instances/111", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PUT", "/v1/osb/999/v2/service_instances/111", invalidBearerAuthHeader},

			{"Missing authorization header", "PATCH", "/v1/osb/999/v2/service_instances/111", emptyAuthHeader},
			{"Invalid authorization schema", "PATCH", "/v1/osb/999/v2/service_instances/111", invalidBasicAuthHeader},
			{"Missing token in authorization header", "PATCH", "/v1/osb/999/v2/service_instances/111", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PATCH", "/v1/osb/999/v2/service_instances/111", invalidBearerAuthHeader},

			{"Missing authorization header", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", emptyAuthHeader},
			{"Invalid authorization schema", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", invalidBasicAuthHeader},
			{"Missing token in authorization header", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", invalidBearerAuthHeader},

			// SERVICE OFFERINGS
			{"Missing authorization header", "GET", web.ServiceOfferingsURL + "/999", emptyAuthHeader},
			{"Invalid basic credentials", "GET", web.ServiceOfferingsURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceOfferingsURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceOfferingsURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.ServiceOfferingsURL, emptyAuthHeader},
			{"Invalid basic credentials", "GET", web.ServiceOfferingsURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceOfferingsURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceOfferingsURL, invalidBearerAuthHeader},

			// SERVICE PLANS
			{"Missing authorization header", "GET", web.ServicePlansURL + "/999", emptyAuthHeader},
			{"Invalid basic credentials", "GET", web.ServicePlansURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServicePlansURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServicePlansURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.ServicePlansURL, emptyAuthHeader},
			{"Invalid basic credentials", "GET", web.ServicePlansURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServicePlansURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServicePlansURL, invalidBearerAuthHeader},

			// VISIBILITIES
			{"Missing authorization header", "GET", web.VisibilitiesURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.VisibilitiesURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.VisibilitiesURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.VisibilitiesURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.VisibilitiesURL + "/999/operations/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.VisibilitiesURL + "/999/operations/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.VisibilitiesURL + "/999/operations/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.VisibilitiesURL + "/999/operations/999", invalidBearerAuthHeader},
			{name: "Valid token in authorization header", method: "GET", path: web.VisibilitiesURL + "/999/operations/999", authHeader: validBasicAuthHeader},

			{"Missing authorization header", "GET", web.VisibilitiesURL, emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.VisibilitiesURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.VisibilitiesURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.VisibilitiesURL, invalidBearerAuthHeader},

			{"Missing authorization header", "POST", web.VisibilitiesURL, emptyAuthHeader},
			{"Invalid authorization schema", "POST", web.VisibilitiesURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "POST", web.VisibilitiesURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "POST", web.VisibilitiesURL, invalidBearerAuthHeader},

			{"Missing authorization header", "PATCH", web.VisibilitiesURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "PATCH", web.VisibilitiesURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "PATCH", web.VisibilitiesURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PATCH", web.VisibilitiesURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "DELETE", web.VisibilitiesURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "DELETE", web.VisibilitiesURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "DELETE", web.VisibilitiesURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "DELETE", web.VisibilitiesURL + "/999", invalidBearerAuthHeader},

			// BROKER PLATFORM CREDENTIALS
			{"Missing authorization header", "POST", web.BrokerPlatformCredentialsURL, emptyAuthHeader},
			{"Invalid authorization schema", "POST", web.BrokerPlatformCredentialsURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "POST", web.BrokerPlatformCredentialsURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "POST", web.BrokerPlatformCredentialsURL, invalidBearerAuthHeader},

			{"Missing authorization header", "PATCH", web.BrokerPlatformCredentialsURL, emptyAuthHeader},
			{"Invalid authorization schema", "PATCH", web.BrokerPlatformCredentialsURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "PATCH", web.BrokerPlatformCredentialsURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PATCH", web.BrokerPlatformCredentialsURL, invalidBearerAuthHeader},

			// CONFIG
			{"Missing authorization header", "GET", "/v1/config", emptyAuthHeader},
			{"Invalid authorization schema", "GET", "/v1/config", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", "/v1/config", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", "/v1/config", invalidBearerAuthHeader},

			// LOGGING CONFIG
			{"Missing authorization header", "GET", web.LoggingConfigURL, emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.LoggingConfigURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.LoggingConfigURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.LoggingConfigURL, invalidBearerAuthHeader},

			{"Missing authorization header", "PUT", web.LoggingConfigURL, emptyAuthHeader},
			{"Invalid authorization schema", "PUT", web.LoggingConfigURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "PUT", web.LoggingConfigURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PUT", web.LoggingConfigURL, invalidBearerAuthHeader},

			// SERVICE INSTANCES
			{"Missing authorization header", "GET", web.ServiceInstancesURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceInstancesURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceInstancesURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceInstancesURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.ServiceInstancesURL + "/999/operations/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceInstancesURL + "/999/operations/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceInstancesURL + "/999/operations/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceInstancesURL + "/999/operations/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.ServiceInstancesURL, emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceInstancesURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceInstancesURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceInstancesURL, invalidBearerAuthHeader},

			{"Missing authorization header", "POST", web.ServiceInstancesURL, emptyAuthHeader},
			{"Invalid authorization schema", "POST", web.ServiceInstancesURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "POST", web.ServiceInstancesURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "POST", web.ServiceInstancesURL, invalidBearerAuthHeader},

			{"Missing authorization header", "PATCH", web.ServiceInstancesURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "PATCH", web.ServiceInstancesURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "PATCH", web.ServiceInstancesURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "PATCH", web.ServiceInstancesURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "DELETE", web.ServiceInstancesURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "DELETE", web.ServiceInstancesURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "DELETE", web.ServiceInstancesURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "DELETE", web.ServiceInstancesURL + "/999", invalidBearerAuthHeader},

			// SERVICE BINDINGS
			{"Missing authorization header", "GET", web.ServiceBindingsURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceBindingsURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceBindingsURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceBindingsURL + "/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.ServiceBindingsURL + "/999/operations/999", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceBindingsURL + "/999/operations/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceBindingsURL + "/999/operations/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceBindingsURL + "/999/operations/999", invalidBearerAuthHeader},

			{"Missing authorization header", "GET", web.ServiceBindingsURL, emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ServiceBindingsURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ServiceBindingsURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ServiceBindingsURL, invalidBearerAuthHeader},

			{"Missing authorization header", "POST", web.ServiceBindingsURL, emptyAuthHeader},
			{"Invalid authorization schema", "POST", web.ServiceBindingsURL, invalidBasicAuthHeader},
			{"Missing token in authorization header", "POST", web.ServiceBindingsURL, emptyBearerAuthHeader},
			{"Invalid token in authorization header", "POST", web.ServiceBindingsURL, invalidBearerAuthHeader},

			{"Missing authorization header", "DELETE", web.ServiceBindingsURL + "/999", emptyAuthHeader},
			{"Invalid authorization schema", "DELETE", web.ServiceBindingsURL + "/999", invalidBasicAuthHeader},
			{"Missing token in authorization header", "DELETE", web.ServiceBindingsURL + "/999", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "DELETE", web.ServiceBindingsURL + "/999", invalidBearerAuthHeader},

			// PROFILE
			{"Missing authorization header", "GET", web.ProfileURL + "/heap", emptyAuthHeader},
			{"Invalid authorization schema", "GET", web.ProfileURL + "/heap", invalidBasicAuthHeader},
			{"Missing token in authorization header", "GET", web.ProfileURL + "/heap", emptyBearerAuthHeader},
			{"Invalid token in authorization header", "GET", web.ProfileURL + "/heap", invalidBearerAuthHeader},
		}

		for _, request := range authRequests {
			request := request
			It(request.name+" "+request.method+" "+request.path, func() {
				expectUnauthorizedRequest(ctx, request.method, request.path, request.authHeader())
			})
		}
	})
})
