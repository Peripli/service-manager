package security_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/env"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/security/authenticators"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

func TestSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Security Tests Suite")
}

type testCase struct {
	path                            string
	method                          string
	noAuthRequestExpectedStatus     int
	basicAuthRequestExpectedStatus  int
	bearerAuthRequestExpectedStatus int
}

var _ = Describe("Service Manager Security Tests", func() {

	var (
		ctx            *common.TestContext
		contextBuilder *common.TestContextBuilder
	)

	BeforeEach(func() {
		contextBuilder = common.NewTestContextBuilder().WithBasicAuthPlatformName("security-tests-platform")
	})

	JustBeforeEach(func() {
		ctx = contextBuilder.Build()
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	describeTable := func(description string, entries []TableEntry) {
		DescribeTable(description, func(t testCase) {
			By("requesting with no auth")
			ctx.SM.Request(t.method, t.path).Expect().Status(t.noAuthRequestExpectedStatus)
			By("requesting with basic auth")
			ctx.SMWithBasic.Request(t.method, t.path).Expect().Status(t.basicAuthRequestExpectedStatus)
			By("requesting with bearer auth")
			ctx.SMWithOAuth.Request(t.method, t.path).Expect().Status(t.bearerAuthRequestExpectedStatus)
		}, entries)
	}

	Describe("Required", func() {
		BeforeEach(func() {
			contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				smb.Security().Path(web.MonitorHealthURL).Method(http.MethodGet).
					Authentication().Required()
				return nil
			})
		})

		entries := []TableEntry{
			Entry("should return 401", testCase{
				path:                            web.MonitorHealthURL,
				method:                          http.MethodGet,
				noAuthRequestExpectedStatus:     http.StatusUnauthorized,
				basicAuthRequestExpectedStatus:  http.StatusUnauthorized,
				bearerAuthRequestExpectedStatus: http.StatusUnauthorized,
			}),
			Entry("should return 200 for path without authentication", testCase{
				path:                            web.PlatformsURL,
				method:                          http.MethodGet,
				noAuthRequestExpectedStatus:     http.StatusOK,
				basicAuthRequestExpectedStatus:  http.StatusOK,
				bearerAuthRequestExpectedStatus: http.StatusOK,
			}),
		}

		describeTable("without explicit authentication type", entries)

		Context("with basic authenticator", func() {
			BeforeEach(func() {
				attachRequiredBasic(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
			})

			entries := []TableEntry{
				Entry("should work for basic auth", testCase{
					path:                            web.MonitorHealthURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusUnauthorized,
					basicAuthRequestExpectedStatus:  http.StatusOK,
					bearerAuthRequestExpectedStatus: http.StatusUnauthorized,
				}),
				Entry("should return 200 for path without authentication", testCase{
					path:                            web.PlatformsURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusOK,
					basicAuthRequestExpectedStatus:  http.StatusOK,
					bearerAuthRequestExpectedStatus: http.StatusOK,
				}),
			}

			describeTable("", entries)
		})

		Context("with bearer authenticator", func() {
			BeforeEach(func() {
				attachRequiredBearer(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet})
			})

			entries := []TableEntry{
				Entry("should work for bearer auth", testCase{
					path:                            web.MonitorHealthURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusUnauthorized,
					basicAuthRequestExpectedStatus:  http.StatusUnauthorized,
					bearerAuthRequestExpectedStatus: http.StatusOK,
				}),
				Entry("should return 200 for path without authentication", testCase{
					path:                            web.PlatformsURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusOK,
					basicAuthRequestExpectedStatus:  http.StatusOK,
					bearerAuthRequestExpectedStatus: http.StatusOK,
				}),
			}

			describeTable("", entries)

			Context("with authorization", func() {
				BeforeEach(func() {
					attachRequiredBearer(contextBuilder, []string{web.PlatformsURL}, []string{http.MethodGet})

					contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
						smb.Security().
							Path(web.MonitorHealthURL).Method(http.MethodGet).
							WithScopes("read_health").Required()

						smb.Security().Path(web.PlatformsURL).Method(http.MethodGet).
							WithClientIDSuffixes([]string{"trustedsuffix", "anothertrustedsfx"}).Required()
						return nil
					})
				})

				Context("no scopes", func() {
					It("should return 403", func() {
						ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusForbidden).
							JSON().Path("$.description").String().
							Contains("none of the scopes [read_health] are present in the user token scopes []")
					})
				})

				Context("with NOT trusted client id suffix", func() {
					BeforeEach(func() {
						contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
							"cid": "not_trusted_id",
						})
					})

					It("should NOT allow access", func() {
						ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().
							Status(http.StatusForbidden).
							JSON().Path("$.description").String().Contains("client id \"not_trusted_id\" from user token does not have the required suffix")
					})
				})

				Context("with trusted client id suffix", func() {
					BeforeEach(func() {
						contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
							"cid": "some_trustedsuffix",
						})
					})
					It("should allow access", func() {
						ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().
							Status(http.StatusOK)
					})
				})

				Context("with another trusted client id suffix", func() {
					BeforeEach(func() {
						contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
							"cid": "some_anothertrustedsfx",
						})
					})
					It("should allow access", func() {
						ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().
							Status(http.StatusOK)
					})
				})

				Context("with scopes", func() {
					BeforeEach(func() {
						contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
							"scope": []string{"read_health"},
						})
					})
					It("should return 200", func() {
						ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
					})
				})

				Context("when multiple authorizators are attached to one path", func() {
					BeforeEach(func() {
						contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
							smb.Security().
								Path(web.MonitorHealthURL).Method(http.MethodGet).
								WithScopes("other_scope").WithClientID("trusted_client").Required()
							return nil
						})
					})

					When("wrong token is used", func() {
						It("should not allow access", func() {
							ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().
								Status(http.StatusForbidden).
								JSON().Path("$.description").String().
								Contains("(cause: none of the scopes [read_health] are present in the user token scopes []) or (cause: none of the scopes [other_scope] are present in the user token scopes []; cause: client id \"\" from user token is not trusted)")
						})
					})
					When("and token with client_id 'trusted_client' and scope 'other_scope' is used", func() {
						BeforeEach(func() {
							contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
								"cid":   "trusted_client",
								"scope": []string{"other_scope"},
							})
						})
						It("should allow to access endpoint", func() {
							ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
						})
					})

					When("and token scope 'read_health'", func() {
						BeforeEach(func() {
							contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
								"scope": []string{"read_health"},
							})
						})
						It("should allow to access endpoint", func() {
							ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
						})
					})
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
							smb.Security().Path(web.MonitorHealthURL).Method(http.MethodGet).
								WithScopes("random").SetAccessLevel(web.TenantAccess).Required()
							return nil
						})
					})

					It("should be set accordingly", func() {
						ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK).Body().Equal(web.TenantAccess.String())
					})
				})

				When("access level is NOT explicitly set", func() {
					BeforeEach(func() {
						contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
							smb.Security().Path(web.MonitorHealthURL).Method(http.MethodGet).
								WithScopes("random").Required()
							return nil
						})
					})

					It("should use default access level", func() {
						ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK).Body().Equal(web.GlobalAccess.String())
					})
				})
			})
		})

		Context("with basic and bearer", func() {
			BeforeEach(func() {
				attachRequiredBasic(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
				attachRequiredBearer(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet})
			})

			entries := []TableEntry{
				Entry("should work for both bearer and basic", testCase{
					path:                            web.MonitorHealthURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusUnauthorized,
					basicAuthRequestExpectedStatus:  http.StatusOK,
					bearerAuthRequestExpectedStatus: http.StatusOK,
				}),
				Entry("should return 200 for path without authentication", testCase{
					path:                            web.PlatformsURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusOK,
					basicAuthRequestExpectedStatus:  http.StatusOK,
					bearerAuthRequestExpectedStatus: http.StatusOK,
				}),
			}

			describeTable("", entries)
		})
	})

	Describe("Optional", func() {
		BeforeEach(func() {
			contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				smb.Security().Path(web.MonitorHealthURL).Method(http.MethodGet).
					Authentication().Optional()
				return nil
			})
		})

		entries := []TableEntry{
			Entry("should return 200", testCase{
				path:                            web.MonitorHealthURL,
				method:                          http.MethodGet,
				noAuthRequestExpectedStatus:     http.StatusOK,
				basicAuthRequestExpectedStatus:  http.StatusOK,
				bearerAuthRequestExpectedStatus: http.StatusOK,
			}),
			Entry("should return for path without authentication", testCase{
				path:                            web.PlatformsURL,
				method:                          http.MethodGet,
				noAuthRequestExpectedStatus:     http.StatusOK,
				basicAuthRequestExpectedStatus:  http.StatusOK,
				bearerAuthRequestExpectedStatus: http.StatusOK,
			}),
		}

		describeTable("without explicit authentication type", entries)

		Context("with authenticator", func() {
			BeforeEach(func() {
				attachOptionalBasic(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
			})

			entries := []TableEntry{
				Entry("should return 200", testCase{
					path:                            web.MonitorHealthURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusOK,
					basicAuthRequestExpectedStatus:  http.StatusOK,
					bearerAuthRequestExpectedStatus: http.StatusOK,
				}),
				Entry("should return 200 for path without authentication", testCase{
					path:                            web.PlatformsURL,
					method:                          http.MethodGet,
					noAuthRequestExpectedStatus:     http.StatusOK,
					basicAuthRequestExpectedStatus:  http.StatusOK,
					bearerAuthRequestExpectedStatus: http.StatusOK,
				}),
			}

			describeTable("", entries)
		})

		Context("with authorization", func() {
			BeforeEach(func() {
				attachRequiredBearer(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet})

				contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
					smb.Security().
						Path(web.MonitorHealthURL).Method(http.MethodGet).
						WithScopes("read_health").Optional()
					return nil
				})
			})

			Context("no scopes", func() {
				It("should return 200", func() {
					ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
				})
			})

			Context("with scopes", func() {
				BeforeEach(func() {
					contextBuilder.WithDefaultTokenClaims(map[string]interface{}{
						"scope": []string{"read_health"},
					})
				})
				It("should return 200", func() {
					ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
				})
			})
		})
	})

	Describe("Mixed", func() {
		Context("required and optional", func() {
			Context("a path is required and subpath is optional", func() {
				BeforeEach(func() {
					attachRequiredBasic(contextBuilder, []string{"/v1/**"}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
					attachOptionalBasic(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
				})

				Context("when requesting required subpath", func() {
					It("should return 401 for no auth", func() {
						ctx.SM.GET(web.PlatformsURL).Expect().Status(http.StatusUnauthorized)
					})

					It("should return 200 for basic", func() {
						ctx.SMWithBasic.GET(web.PlatformsURL).Expect().Status(http.StatusOK)
					})
				})

				Context("when requesting optional subpath", func() {
					It("should return 200 for no auth", func() {
						ctx.SM.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
					})

					It("should return 200 for basic", func() {
						ctx.SMWithBasic.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
					})
				})
			})

			Context("a path is optional and subpath is required", func() {
				BeforeEach(func() {
					attachRequiredBasic(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
					attachOptionalBasic(contextBuilder, []string{"/v1/**"}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
				})

				Context("when requesting required subpath", func() {
					It("should return 200 for no auth", func() {
						ctx.SM.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
					})

					It("should return 200 for basic", func() {
						ctx.SMWithBasic.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
					})
				})
			})

			Context("with authenticator for GET and POST", func() {
				BeforeEach(func() {
					attachRequiredBasic(contextBuilder, []string{web.ServiceInstancesURL}, []string{http.MethodPost}, authenticators.BasicPlatformAuthenticator)
					attachOptionalBasic(contextBuilder, []string{web.ServiceInstancesURL}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
				})

				It("should return 401 for POST with no auth", func() {
					ctx.SM.POST(web.ServiceInstancesURL).WithJSON("").Expect().Status(http.StatusUnauthorized)
				})

				It("should NOT return 401 for POST with auth", func() {
					Expect(ctx.SM.POST(web.ServiceInstancesURL).Expect().Raw().StatusCode).NotTo(Equal(http.StatusUnauthorized))
				})

				It("should return 200 for GET", func() {
					ctx.SM.GET(web.ServiceInstancesURL).Expect().Status(http.StatusOK)
				})
			})

			Context("basic required and bearer optional", func() {
				BeforeEach(func() {
					attachRequiredBasic(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet}, authenticators.BasicPlatformAuthenticator)
					attachOptionalBearer(contextBuilder, []string{web.MonitorHealthURL}, []string{http.MethodGet})

					contextBuilder.WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
						smb.RegisterFilters(&testFilter{
							middlewareFunc: func(req *web.Request, next web.Handler) (*web.Response, error) {
								user, found := web.UserFromContext(req.Context())
								if !found {
									return &web.Response{
										StatusCode: http.StatusOK,
										Body:       []byte("NoAuth"),
									}, nil
								}
								return &web.Response{
									StatusCode: http.StatusOK,
									Body:       []byte(user.AuthenticationType.String()),
								}, nil
							},
						})
						return nil
					})
				})

				It("should return NoAuth with no auth", func() {
					ctx.SM.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK).Body().Equal("NoAuth")
				})

				It("should return Basic with basic auth", func() {
					ctx.SMWithBasic.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK).Body().Equal(web.Basic.String())
				})

				It("should return Bearer with bearer auth", func() {
					ctx.SMWithOAuth.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK).Body().Equal(web.Bearer.String())
				})
			})
		})

		Context("with authorization", func() {
		})
	})
})

func newBearerAuthenticator(issuerURL, clientID string) httpsec.Authenticator {
	authenticator, _, err := authenticators.NewOIDCAuthenticator(context.Background(), &authenticators.OIDCOptions{
		IssuerURL: issuerURL,
		ClientID:  clientID,
	})
	Expect(err).ShouldNot(HaveOccurred())
	return authenticator
}

func attachBearer(contextBuilder *common.TestContextBuilder, paths []string, methods []string, required bool) {
	contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
		secState := smb.Security().Path(paths...).Method(methods...).
			WithAuthentication(newBearerAuthenticator(e.Get("api.token_issuer_url").(string), e.Get("api.client_id").(string)))
		if required {
			secState.Required()
		} else {
			secState.Optional()
		}
		return nil
	})
}

func attachRequiredBearer(contextBuilder *common.TestContextBuilder, paths []string, methods []string) {
	attachBearer(contextBuilder, paths, methods, true)
}

func attachOptionalBearer(contextBuilder *common.TestContextBuilder, paths []string, methods []string) {
	attachBearer(contextBuilder, paths, methods, false)
}

func attachRequiredBasic(contextBuilder *common.TestContextBuilder, paths []string, methods []string, authenticatorFunc authenticators.BasicAuthenticatorFunc) {
	attachBasic(contextBuilder, paths, methods, authenticatorFunc, true)
}

func attachOptionalBasic(contextBuilder *common.TestContextBuilder, paths []string, methods []string, authenticatorFunc authenticators.BasicAuthenticatorFunc) {
	attachBasic(contextBuilder, paths, methods, authenticatorFunc, false)
}

func attachBasic(contextBuilder *common.TestContextBuilder, paths []string, methods []string, authenticatorFunc authenticators.BasicAuthenticatorFunc, required bool) {
	contextBuilder.WithSMExtensions(func(_ context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
		secState := smb.Security().Path(paths...).Method(methods...).
			WithAuthentication(&authenticators.Basic{
				Repository:             smb.Storage,
				BasicAuthenticatorFunc: authenticatorFunc,
			})
		if required {
			secState.Required()
		} else {
			secState.Optional()
		}
		return nil
	})
}

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
		{
			Matchers: []web.Matcher{
				web.Path(web.MonitorHealthURL),
			},
		},
	}
}
