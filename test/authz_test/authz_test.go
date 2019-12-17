package authz_test

import (
	"context"
	"net/http"
	"testing"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/security/authenticators"

	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuthentication(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authentication Tests Suite")
}

var _ = Describe("Service Manager Authentication", func() {

	authenticatorFunc := func(issuerURL, clientID string) httpsec.Authenticator {
		authenticator, _, err := authenticators.NewOIDCAuthenticator(context.Background(), &authenticators.OIDCOptions{
			IssuerURL: issuerURL,
			ClientID:  clientID,
		})
		Expect(err).ShouldNot(HaveOccurred())
		return authenticator
	}

	var (
		ctx            *common.TestContext
		contextBuilder *common.TestContextBuilder
	)

	BeforeEach(func() {
		contextBuilder = common.NewTestContextBuilder()
	})

	JustBeforeEach(func() {
		ctx = contextBuilder.Build()
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	When("optional subpath", func() {
		BeforeEach(func() {
			contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				smb.Security().Path("/**").Method(http.MethodGet).
					WithAuthentication(&filters.BasicAuthenticator{
						Repository: smb.Storage,
					}).
					WithAuthentication(authenticatorFunc(e.Get("api.token_issuer_url").(string), e.Get("api.client_id").(string))).
					Required()

				smb.Security().Path("/v1/monitor/health").Method(http.MethodGet).
					Authentication().
					Optional()
				return nil
			})
		})

		It("should not require health authentication", func() {
			ctx.SM.GET("/v1/monitor/health").Expect().Status(http.StatusOK)
		})

		It("should require broker authentication", func() {
			ctx.SM.GET("/v1/service_brokers").Expect().Status(http.StatusUnauthorized)
			ctx.SMWithBasic.GET("/v1/service_brokers").Expect().Status(http.StatusOK)
		})

		When("authentication is not ok", func() {
			It("should return understandable error", func() {
				ctx.SM.GET("/v1/service_brokers").WithHeader("Authorization", "Basic not-ok").Expect().Status(http.StatusUnauthorized).Body()
			})
		})

		When("with scopes authorization", func() {
			BeforeEach(func() {
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithScopes("read").Required()
					return nil
				})
			})

			It("should return understandable error", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").Expect().
					Status(http.StatusForbidden).
					JSON().Path("$.description").String().
					Contains("none of the scopes [read] are present in the user token scopes []")
			})
		})

		When("with client id authorization", func() {
			BeforeEach(func() {
				contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
					"cid": "not_trusted_id",
				})
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithClientID("myid").Required()
					return nil
				})
			})

			It("should return understandable error", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").Expect().
					Status(http.StatusForbidden).
					JSON().Path("$.description").String().Contains("client id \"not_trusted_id\" from user token is not trusted")
			})
		})

		When("with client id suffix authorization", func() {
			BeforeEach(func() {
				contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
					"cid": "not_trusted_id",
				})
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithClientIDSuffix("trustedsuffix").Required()
					return nil
				})
			})

			It("should return understandable error", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").Expect().
					Status(http.StatusForbidden).
					JSON().Path("$.description").String().Contains("client id \"not_trusted_id\" from user token does not have the required suffix")
			})
		})

		When("with multiple and authorizations", func() {
			BeforeEach(func() {
				contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
					"cid": "not_trusted_id",
				})
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithClientIDSuffix("trustedsuffix").WithScopes("read").Required()
					return nil
				})
			})

			It("should return understandable error", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").Expect().
					Status(http.StatusForbidden).
					JSON().Path("$.description").String().
					Contains("client id \"not_trusted_id\" from user token does not have the required suffix").
					Contains("none of the scopes [read] are present in the user token scopes []")
			})
		})

		When("with multiple or authorizations", func() {
			BeforeEach(func() {
				contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
					"cid": "not_trusted_id",
				})
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithClientIDSuffix("trustedsuffix").WithScopes("random").Required().
						WithScopes("read").Required()
					return nil
				})
			})

			It("should return understandable error", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").Expect().
					Status(http.StatusForbidden).
					JSON().Path("$.description").String().
					Contains("(cause: client id \"not_trusted_id\" from user token does not have the required suffix; cause: none of the scopes [random] are present in the user token scopes []) or (cause: none of the scopes [read] are present in the user token scopes [])")
			})
		})

		Context("access level tests", func() {
			BeforeEach(func() {
				contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
					"scope": []string{"random"},
				})

				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.RegisterFilters(&testFilter{
						middlewareFunc: func(req *web.Request, next web.Handler) (*web.Response, error) {
							user, _ := web.UserFromContext(req.Context())
							return &web.Response{
								StatusCode: http.StatusOK,
								Body:       []byte(user.AccessLevel.String()),
							}, nil
						},
					})
					return nil
				})
			})

			When("access level is explicitly set", func() {
				BeforeEach(func() {
					contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
						smb.Security().Path("/**").Method(http.MethodGet).
							WithScopes("random").SetAccessLevel(web.TenantAccess).Required()
						return nil
					})
				})

				It("should be set accordingly", func() {
					ctx.SMWithOAuth.GET("/v1/info").Expect().Status(http.StatusOK).Body().Equal(web.TenantAccess.String())
				})
			})

			When("access level is NOT explicitly set", func() {
				BeforeEach(func() {
					contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
						smb.Security().Path("/**").Method(http.MethodGet).
							WithScopes("random").Required()
						return nil
					})
				})

				It("should use default access level", func() {
					ctx.SMWithOAuth.GET("/v1/info").Expect().Status(http.StatusOK).Body().Equal(web.GlobalAccess.String())
				})
			})
		})
	})

	When("two types of authentication are attached", func() {
		When("optional", func() {
			BeforeEach(func() {
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithAuthentication(&filters.BasicAuthenticator{
							Repository: smb.Storage,
						}).
						WithAuthentication(authenticatorFunc(e.Get("api.token_issuer_url").(string), e.Get("api.client_id").(string))).
						Optional()
					return nil
				})
			})

			Context("with no auth", func() {
				It("should return 200", func() {
					ctx.SM.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				})
			})

			Context("with basic auth", func() {
				It("should return 200", func() {
					ctx.SMWithBasic.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				})
			})

			Context("with oauth", func() {
				It("should return 200", func() {
					ctx.SMWithOAuth.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				})
			})
		})

		When("required", func() {
			BeforeEach(func() {
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithAuthentication(&filters.BasicAuthenticator{
							Repository: smb.Storage,
						}).
						WithAuthentication(authenticatorFunc(e.Get("api.token_issuer_url").(string), e.Get("api.client_id").(string))).
						Required()
					return nil
				})
			})

			Context("with no auth", func() {
				It("should return 401", func() {
					ctx.SM.GET(web.ServiceBrokersURL).Expect().Status(http.StatusUnauthorized)
				})
			})

			Context("with basic auth", func() {
				It("should return 200", func() {
					ctx.SMWithBasic.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				})
			})

			Context("with oauth", func() {
				It("should return 200", func() {
					ctx.SMWithOAuth.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				})
			})
		})
	})

	When("it is optional", func() {
		BeforeEach(func() {
			contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				smb.Security().Path("/**").Method(http.MethodGet).WithAuthentication(&filters.BasicAuthenticator{
					Repository: smb.Storage,
				}).Optional()
				return nil
			})
		})

		When("and using basic auth", func() {
			It("returns 200", func() {
				ctx.SMWithBasic.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				ctx.SMWithBasic.GET(web.PlatformsURL).Expect().Status(http.StatusOK)
			})
		})

		When("and using no auth", func() {
			It("returns 200", func() {
				ctx.SM.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				ctx.SM.GET(web.PlatformsURL).Expect().Status(http.StatusOK)
			})
		})

		Context("with another required for same path", func() {
			BeforeEach(func() {
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path("/**").Method(http.MethodGet).
						WithAuthentication(authenticatorFunc(e.Get("api.token_issuer_url").(string), e.Get("api.client_id").(string))).
						Required()
					return nil
				})
			})

			When("no auth is used", func() {
				It("returns 401", func() {
					ctx.SM.GET(web.ServiceBrokersURL).Expect().Status(http.StatusUnauthorized)
				})
			})

			When("using basic auth", func() {
				It("returns 200", func() {
					ctx.SMWithBasic.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				})
			})

			When("using bearer auth", func() {
				It("returns 200", func() {
					ctx.SMWithOAuth.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
				})
			})
		})

		Context("with another required for subpath", func() {
			BeforeEach(func() {
				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().Path(web.ServiceBrokersURL).Method(http.MethodGet).WithAuthentication(&filters.BasicAuthenticator{
						Repository: smb.Storage,
					}).Required()
					return nil
				})
			})

			When("and using basic auth", func() {
				It("returns 200", func() {
					ctx.SMWithBasic.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
					ctx.SMWithBasic.GET(web.PlatformsURL).Expect().Status(http.StatusOK)
				})
			})

			When("and using no auth", func() {
				It("returns 200", func() {
					ctx.SM.GET(web.ServiceBrokersURL).Expect().Status(http.StatusUnauthorized)
					ctx.SM.GET(web.PlatformsURL).Expect().Status(http.StatusOK)
				})
			})
		})
	})
})

type testFilter struct {
	middlewareFunc web.MiddlewareFunc
}

func (tf *testFilter) Name() string {
	return "testFilter"
}

func (tf *testFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	return tf.middlewareFunc(req, next)
}

func (tf *testFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/v1/info"),
			},
		},
	}
}
