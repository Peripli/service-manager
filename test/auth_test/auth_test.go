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
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
	"testing"

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
		authRequests := []struct{ name, method, path, authHeader string }{
			// PLATFORMS
			{"Missing authorization header", "GET", web.PlatformsURL + "/999", ""},
			{"Invalid authorization schema", "GET", web.PlatformsURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "GET", web.PlatformsURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "GET", web.PlatformsURL + "/999", "Bearer abc"},

			{"Missing authorization header", "GET", web.PlatformsURL, ""},
			{"Invalid authorization schema", "GET", web.PlatformsURL, "Basic abc"},
			{"Missing token in authorization header", "GET", web.PlatformsURL, "Bearer "},
			{"Invalid token in authorization header", "GET", web.PlatformsURL, "Bearer abc"},

			{"Missing authorization header", "POST", web.PlatformsURL, ""},
			{"Invalid authorization schema", "POST", web.PlatformsURL, "Basic abc"},
			{"Missing token in authorization header", "POST", web.PlatformsURL, "Bearer "},
			{"Invalid token in authorization header", "POST", web.PlatformsURL, "Bearer abc"},

			{"Missing authorization header", "PATCH", web.PlatformsURL + "/999", ""},
			{"Invalid authorization schema", "PATCH", web.PlatformsURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "PATCH", web.PlatformsURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "PATCH", web.PlatformsURL + "/999", "Bearer abc"},

			{"Missing authorization header", "DELETE", web.PlatformsURL + "/999", ""},
			{"Invalid authorization schema", "DELETE", web.PlatformsURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "DELETE", web.PlatformsURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "DELETE", web.PlatformsURL + "/999", "Bearer abc"},

			// BROKERS
			{"Missing authorization header", "GET", web.ServiceBrokersURL + "/999", ""},
			{"Invalid authorization schema", "GET", web.ServiceBrokersURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "GET", web.ServiceBrokersURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "GET", web.ServiceBrokersURL + "/999", "Bearer abc"},

			{"Missing authorization header", "GET", web.ServiceBrokersURL, ""},
			{"Invalid authorization schema", "GET", web.ServiceBrokersURL, "Basic abc"},
			{"Missing token in authorization header", "GET", web.ServiceBrokersURL, "Bearer "},
			{"Invalid token in authorization header", "GET", web.ServiceBrokersURL, "Bearer abc"},

			{"Missing authorization header", "POST", web.ServiceBrokersURL, ""},
			{"Invalid authorization schema", "POST", web.ServiceBrokersURL, "Basic abc"},
			{"Missing token in authorization header", "POST", web.ServiceBrokersURL, "Bearer "},
			{"Invalid token in authorization header", "POST", web.ServiceBrokersURL, "Bearer abc"},

			{"Missing authorization header", "PATCH", web.ServiceBrokersURL + "/999", ""},
			{"Invalid authorization schema", "PATCH", web.ServiceBrokersURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "PATCH", web.ServiceBrokersURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "PATCH", web.ServiceBrokersURL + "/999", "Bearer abc"},

			{"Missing authorization header", "DELETE", web.ServiceBrokersURL + "/999", ""},
			{"Invalid authorization schema", "DELETE", web.ServiceBrokersURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "DELETE", web.ServiceBrokersURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "DELETE", web.ServiceBrokersURL + "/999", "Bearer abc"},

			// OSB
			{"Missing authorization header", "GET", "/v1/osb/999/v2/catalog", ""},
			{"Invalid authorization schema", "GET", "/v1/osb/999/v2/catalog", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/osb/999/v2/catalog", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/osb/999/v2/catalog", "Bearer abc"},

			{"Missing authorization header", "PUT", "/v1/osb/999/v2/service_instances/111", ""},
			{"Invalid authorization schema", "PUT", "/v1/osb/999/v2/service_instances/111", "Basic abc"},
			{"Missing token in authorization header", "PUT", "/v1/osb/999/v2/service_instances/111", "Bearer "},
			{"Invalid token in authorization header", "PUT", "/v1/osb/999/v2/service_instances/111", "Bearer abc"},

			{"Missing authorization header", "PATCH", "/v1/osb/999/v2/service_instances/111", ""},
			{"Invalid authorization schema", "PATCH", "/v1/osb/999/v2/service_instances/111", "Basic abc"},
			{"Missing token in authorization header", "PATCH", "/v1/osb/999/v2/service_instances/111", "Bearer "},
			{"Invalid token in authorization header", "PATCH", "/v1/osb/999/v2/service_instances/111", "Bearer abc"},

			{"Missing authorization header", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", ""},
			{"Invalid authorization schema", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", "Basic abc"},
			{"Missing token in authorization header", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", "Bearer "},
			{"Invalid token in authorization header", "DELETE", "/v1/osb/999/v2/service_instances/111/service_bindings/222", "Bearer abc"},

			// SERVICE OFFERINGS
			{"Missing authorization header", "GET", web.ServiceOfferingsURL + "/999", ""},
			{"Invalid basic credentials", "GET", web.ServiceOfferingsURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "GET", web.ServiceOfferingsURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "GET", web.ServiceOfferingsURL + "/999", "Bearer abc"},

			{"Missing authorization header", "GET", web.ServiceOfferingsURL, ""},
			{"Invalid basic credentials", "GET", web.ServiceOfferingsURL, "Basic abc"},
			{"Missing token in authorization header", "GET", web.ServiceOfferingsURL, "Bearer "},
			{"Invalid token in authorization header", "GET", web.ServiceOfferingsURL, "Bearer abc"},

			// SERVICE PLANS
			{"Missing authorization header", "GET", web.ServicePlansURL + "/999", ""},
			{"Invalid basic credentials", "GET", web.ServicePlansURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "GET", web.ServicePlansURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "GET", web.ServicePlansURL + "/999", "Bearer abc"},

			{"Missing authorization header", "GET", web.ServicePlansURL, ""},
			{"Invalid basic credentials", "GET", web.ServicePlansURL, "Basic abc"},
			{"Missing token in authorization header", "GET", web.ServicePlansURL, "Bearer "},
			{"Invalid token in authorization header", "GET", web.ServicePlansURL, "Bearer abc"},

			// VISIBILITIES
			{"Missing authorization header", "GET", web.VisibilitiesURL + "/999", ""},
			{"Invalid authorization schema", "GET", web.VisibilitiesURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "GET", web.VisibilitiesURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "GET", web.VisibilitiesURL + "/999", "Bearer abc"},

			{"Missing authorization header", "GET", web.VisibilitiesURL, ""},
			{"Invalid authorization schema", "GET", web.VisibilitiesURL, "Basic abc"},
			{"Missing token in authorization header", "GET", web.VisibilitiesURL, "Bearer "},
			{"Invalid token in authorization header", "GET", web.VisibilitiesURL, "Bearer abc"},

			{"Missing authorization header", "POST", web.VisibilitiesURL, ""},
			{"Invalid authorization schema", "POST", web.VisibilitiesURL, "Basic abc"},
			{"Missing token in authorization header", "POST", web.VisibilitiesURL, "Bearer "},
			{"Invalid token in authorization header", "POST", web.VisibilitiesURL, "Bearer abc"},

			{"Missing authorization header", "PATCH", web.VisibilitiesURL + "/999", ""},
			{"Invalid authorization schema", "PATCH", web.VisibilitiesURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "PATCH", web.VisibilitiesURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "PATCH", web.VisibilitiesURL + "/999", "Bearer abc"},

			{"Missing authorization header", "DELETE", web.VisibilitiesURL + "/999", ""},
			{"Invalid authorization schema", "DELETE", web.VisibilitiesURL + "/999", "Basic abc"},
			{"Missing token in authorization header", "DELETE", web.VisibilitiesURL + "/999", "Bearer "},
			{"Invalid token in authorization header", "DELETE", web.VisibilitiesURL + "/999", "Bearer abc"},

			// LOGGING CONFIG
			{"Missing authorization header", "GET", web.LoggingConfigURL, ""},
			{"Invalid authorization schema", "GET", web.LoggingConfigURL, "Basic abc"},
			{"Missing token in authorization header", "GET", web.LoggingConfigURL, "Bearer "},
			{"Invalid token in authorization header", "GET", web.LoggingConfigURL, "Bearer abc"},

			{"Missing authorization header", "PUT", web.LoggingConfigURL, ""},
			{"Invalid authorization schema", "PUT", web.LoggingConfigURL, "Basic abc"},
			{"Missing token in authorization header", "PUT", web.LoggingConfigURL, "Bearer "},
			{"Invalid token in authorization header", "PUT", web.LoggingConfigURL, "Bearer abc"},
		}

		for _, request := range authRequests {
			request := request
			It(request.name+" "+request.method+" "+request.path+" auth_header: "+request.authHeader, func() {
				expectUnauthorizedRequest(ctx, request.method, request.path, request.authHeader)
			})
		}
	})
})
