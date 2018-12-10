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
		ctx = common.NewTestContext(nil)
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	Context("Nontrivial scenarios", func() {
		It("Forbidden broker registration with basic auth", func() {
			ctx.SMWithBasic.GET("/v1/service_brokers").
				Expect().
				Status(http.StatusOK)

			ctx.SMWithBasic.POST("/v1/service_brokers").
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
			{"Missing authorization header", "GET", "/v1/platforms/999", ""},
			{"Invalid authorization schema", "GET", "/v1/platforms/999", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/platforms/999", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/platforms/999", "Bearer abc"},

			{"Missing authorization header", "GET", "/v1/platforms", ""},
			{"Invalid authorization schema", "GET", "/v1/platforms", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/platforms", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/platforms", "Bearer abc"},

			{"Missing authorization header", "POST", "/v1/platforms", ""},
			{"Invalid authorization schema", "POST", "/v1/platforms", "Basic abc"},
			{"Missing token in authorization header", "POST", "/v1/platforms", "Bearer "},
			{"Invalid token in authorization header", "POST", "/v1/platforms", "Bearer abc"},

			{"Missing authorization header", "PATCH", "/v1/platforms/999", ""},
			{"Invalid authorization schema", "PATCH", "/v1/platforms/999", "Basic abc"},
			{"Missing token in authorization header", "PATCH", "/v1/platforms/999", "Bearer "},
			{"Invalid token in authorization header", "PATCH", "/v1/platforms/999", "Bearer abc"},

			{"Missing authorization header", "DELETE", "/v1/platforms/999", ""},
			{"Invalid authorization schema", "DELETE", "/v1/platforms/999", "Basic abc"},
			{"Missing token in authorization header", "DELETE", "/v1/platforms/999", "Bearer "},
			{"Invalid token in authorization header", "DELETE", "/v1/platforms/999", "Bearer abc"},

			// BROKERS
			{"Missing authorization header", "GET", "/v1/service_brokers/999", ""},
			{"Invalid authorization schema", "GET", "/v1/service_brokers/999", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/service_brokers/999", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/service_brokers/999", "Bearer abc"},

			{"Missing authorization header", "GET", "/v1/service_brokers", ""},
			{"Invalid authorization schema", "GET", "/v1/service_brokers", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/service_brokers", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/service_brokers", "Bearer abc"},

			{"Missing authorization header", "POST", "/v1/service_brokers", ""},
			{"Invalid authorization schema", "POST", "/v1/service_brokers", "Basic abc"},
			{"Missing token in authorization header", "POST", "/v1/service_brokers", "Bearer "},
			{"Invalid token in authorization header", "POST", "/v1/service_brokers", "Bearer abc"},

			{"Missing authorization header", "PATCH", "/v1/service_brokers/999", ""},
			{"Invalid authorization schema", "PATCH", "/v1/service_brokers/999", "Basic abc"},
			{"Missing token in authorization header", "PATCH", "/v1/service_brokers/999", "Bearer "},
			{"Invalid token in authorization header", "PATCH", "/v1/service_brokers/999", "Bearer abc"},

			{"Missing authorization header", "DELETE", "/v1/service_brokers/999", ""},
			{"Invalid authorization schema", "DELETE", "/v1/service_brokers/999", "Basic abc"},
			{"Missing token in authorization header", "DELETE", "/v1/service_brokers/999", "Bearer "},
			{"Invalid token in authorization header", "DELETE", "/v1/service_brokers/999", "Bearer abc"},

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
			{"Missing authorization header", "GET", "/v1/service_offerings/999", ""},
			{"Invalid basic credentials", "GET", "/v1/service_offerings/999", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/service_offerings/999", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/service_offerings/999", "Bearer abc"},

			{"Missing authorization header", "GET", "/v1/service_offerings", ""},
			{"Invalid basic credentials", "GET", "/v1/service_offerings", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/service_offerings", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/service_offerings", "Bearer abc"},

			// SERVICE PLANS
			{"Missing authorization header", "GET", "/v1/service_plans/999", ""},
			{"Invalid basic credentials", "GET", "/v1/service_plans/999", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/service_plans/999", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/service_plans/999", "Bearer abc"},

			{"Missing authorization header", "GET", "/v1/service_plans", ""},
			{"Invalid basic credentials", "GET", "/v1/service_plans", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/service_plans", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/service_plans", "Bearer abc"},

			// VISIBILITIES
			{"Missing authorization header", "GET", "/v1/visibilities/999", ""},
			{"Invalid authorization schema", "GET", "/v1/visibilities/999", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/visibilities/999", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/visibilities/999", "Bearer abc"},

			{"Missing authorization header", "GET", "/v1/visibilities", ""},
			{"Invalid authorization schema", "GET", "/v1/visibilities", "Basic abc"},
			{"Missing token in authorization header", "GET", "/v1/visibilities", "Bearer "},
			{"Invalid token in authorization header", "GET", "/v1/visibilities", "Bearer abc"},

			{"Missing authorization header", "POST", "/v1/visibilities", ""},
			{"Invalid authorization schema", "POST", "/v1/visibilities", "Basic abc"},
			{"Missing token in authorization header", "POST", "/v1/visibilities", "Bearer "},
			{"Invalid token in authorization header", "POST", "/v1/visibilities", "Bearer abc"},

			{"Missing authorization header", "PATCH", "/v1/visibilities/999", ""},
			{"Invalid authorization schema", "PATCH", "/v1/visibilities/999", "Basic abc"},
			{"Missing token in authorization header", "PATCH", "/v1/visibilities/999", "Bearer "},
			{"Invalid token in authorization header", "PATCH", "/v1/visibilities/999", "Bearer abc"},

			{"Missing authorization header", "DELETE", "/v1/visibilities/999", ""},
			{"Invalid authorization schema", "DELETE", "/v1/visibilities/999", "Basic abc"},
			{"Missing token in authorization header", "DELETE", "/v1/visibilities/999", "Bearer "},
			{"Invalid token in authorization header", "DELETE", "/v1/visibilities/999", "Bearer abc"},
		}

		for _, request := range authRequests {
			request := request
			It(request.name+" "+request.method+" "+request.path+" auth_header: "+request.authHeader, func() {
				expectUnauthorizedRequest(ctx, request.method, request.path, request.authHeader)
			})
		}
	})
})
