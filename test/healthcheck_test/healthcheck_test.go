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

package healthcheck_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func TestHealth(t *testing.T) {
	RunSpecs(t, "Healthcheck Suite")
}

var _ = Describe("Healthcheck API", func() {

	var (
		ctxBuilder *common.TestContextBuilder
		ctx        *common.TestContext
	)

	BeforeEach(func() {
		ctxBuilder = common.NewTestContextBuilderWithSecurity()
	})

	JustBeforeEach(func() {
		ctx = ctxBuilder.Build()
	})

	Describe("Unauthorized", func() {
		When("Get info handler", func() {
			It("Returns correct response", func() {
				ctx.SM.GET(healthcheck.URL).
					Expect().
					Status(http.StatusOK).JSON().Object().ContainsMap(map[string]interface{}{
					"status": "UP",
				})
			})
		})
	})

	Describe("Authorized", func() {
		When("Get info handler", func() {
			BeforeEach(func() {
				ctxBuilder.WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.RegisterFilters(&dummyAuthFilter{})
					return nil
				})
			})

			It("Doesn't include SM platform", func() {
				respBody := ctx.SMWithOAuth.GET(healthcheck.URL).
					Expect().
					Status(http.StatusOK).JSON().Object()

				respBody.NotContainsMap(map[string]interface{}{
					"details": map[string]interface{}{
						"platforms": map[string]interface{}{
							"details": map[string]interface{}{
								types.SMPlatform: map[string]interface{}{},
							},
						},
					},
				})
			})
		})
	})
})

type dummyAuthFilter struct{}

func (*dummyAuthFilter) Name() string {
	return "dummyAuthFilter"
}

func (*dummyAuthFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := web.ContextWithAuthorization(req.Context())
	req.Request = req.WithContext(ctx)

	return next.Handle(req)
}

func (*dummyAuthFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.MonitorHealthURL),
				web.Methods(http.MethodGet),
			},
		},
	}
}
