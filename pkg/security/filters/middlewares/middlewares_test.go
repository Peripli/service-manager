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

package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/security/http/httpfakes"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMiddlewares(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Security Middlewares Suite")
}

var _ = Describe("Middlewares", func() {
	const expectedErrorMessage = "expected error"
	var (
		req     *web.Request
		handler *webfakes.FakeHandler
	)

	Describe("Authz middleware", func() {
		const filterName = "authzFilterName"
		var authorizer *httpfakes.FakeAuthorizer

		BeforeEach(func() {
			req = &web.Request{
				Request: httptest.NewRequest("GET", "/", nil),
			}
			authorizer = &httpfakes.FakeAuthorizer{}
			handler = &webfakes.FakeHandler{}
		})

		Describe("Run", func() {
			Context("when authorizer returns decision", func() {
				Context("Deny", func() {
					It("should return error", func() {
						authorizer.AuthorizeReturns(httpsec.Deny, web.NoAccess, nil)
						authzFilter := Authorization{
							Authorizer: authorizer,
						}
						_, err := authzFilter.Run(req, handler)
						httpErr, ok := err.(*util.HTTPError)
						Expect(ok).To(BeTrue())
						Expect(httpErr.StatusCode).To(Equal(http.StatusForbidden))
					})
				})
				Context("Abstain", func() {
					It("should continue with calling handler", func() {
						authorizer.AuthorizeReturns(httpsec.Abstain, web.NoAccess, nil)
						handler.HandleReturns(nil, errors.New(expectedErrorMessage))
						authzFilter := Authorization{
							Authorizer: authorizer,
						}
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						Expect(web.IsAuthorized(req.Context())).To(BeFalse())
					})
				})
				Context("Allow", func() {
					Context("when usercontext is missing", func() {
						BeforeEach(func() {
							req.Request = req.WithContext(context.TODO())
						})

						It("should return an error", func() {
							authorizer.AuthorizeReturns(httpsec.Allow, web.GlobalAccess, nil)
							handler.HandleReturns(nil, errors.New(expectedErrorMessage))
							authzFilter := Authorization{
								Authorizer: authorizer,
							}
							_, err := authzFilter.Run(req, handler)
							checkExpectedErrorMessage("authorization failed due to missing user context", err)
							Expect(web.IsAuthorized(req.Context())).To(BeFalse())
						})
					})

					testCase := func(initialLevel, mockLevel, expectedLevel web.AccessLevel, expectedHandlerCallCount int, expectedErrorMessage string, expectAuthrized bool) {
						req.Request = req.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
							Name:               "test-user",
							AccessLevel:        initialLevel,
							AuthenticationType: web.Bearer,
						}))
						authorizer.AuthorizeReturns(httpsec.Allow, mockLevel, nil)
						handler.HandleReturns(nil, errors.New(expectedErrorMessage))
						authzFilter := Authorization{
							Authorizer: authorizer,
						}
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						Expect(web.IsAuthorized(req.Context())).To(Equal(expectAuthrized))
						if expectedHandlerCallCount > 0 {
							req := handler.HandleArgsForCall(0)
							userContext, found := web.UserFromContext(req.Context())
							Expect(found).To(BeTrue())
							Expect(userContext.AccessLevel).To(Equal(expectedLevel))
						}
					}

					Context("when usercontext is present", func() {
						Context("with access level is NoAccess", func() {
							It("should not modify the user context access level", func() {
								testCase(web.GlobalAccess, web.NoAccess, web.GlobalAccess, 0, "authorization failed due to missing access level", false)
							})
						})

						Context("with access level is Global", func() {
							It("should modify the user context access level", func() {
								testCase(web.NoAccess, web.GlobalAccess, web.GlobalAccess, 1, expectedErrorMessage, true)
							})
						})

						Context("with access level is Tenant", func() {
							It("should modify the user context access level", func() {
								testCase(web.NoAccess, web.TenantAccess, web.TenantAccess, 1, expectedErrorMessage, true)
							})
						})

						Context("with access level is AllTenant", func() {
							It("should modify the user context access level", func() {
								testCase(web.NoAccess, web.AllTenantAccess, web.AllTenantAccess, 1, expectedErrorMessage, true)
							})
						})
					})
				})
			})

			Context("when authorizer returns error", func() {
				Context("and decision Abstain", func() {
					It("should return error", func() {
						authorizer.AuthorizeReturns(httpsec.Abstain, web.NoAccess, errors.New(expectedErrorMessage))
						authzFilter := Authorization{
							Authorizer: authorizer,
						}
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
					})
				})

				Context("and decision Deny", func() {
					It("should return http error 403", func() {
						authorizer.AuthorizeReturns(httpsec.Deny, web.NoAccess, errors.New(expectedErrorMessage))
						authzFilter := Authorization{
							Authorizer: authorizer,
						}
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						httpErr, ok := err.(*util.HTTPError)
						Expect(ok).To(BeTrue())
						Expect(httpErr.StatusCode).To(Equal(http.StatusForbidden))
					})
				})
			})
		})

	})

	Describe("Authn middleware", func() {
		var authenticator *httpfakes.FakeAuthenticator

		BeforeEach(func() {
			req = &web.Request{
				Request: httptest.NewRequest("GET", "/", nil),
			}
			authenticator = &httpfakes.FakeAuthenticator{}
			handler = &webfakes.FakeHandler{}
		})

		Describe("Run", func() {
			Context("when authentication already passed", func() {
				It("should continue", func() {
					authnFilter := Authentication{
						Authenticator: nil,
					}
					req.Request = req.Request.WithContext(web.ContextWithUser(req.Context(), &web.UserContext{}))
					authnFilter.Run(req, handler)
					Expect(handler.HandleCallCount()).To(Equal(1))
				})
			})

			Context("when authenticator returns decision", func() {
				Context("Deny", func() {
					It("should return error", func() {
						authenticator.AuthenticateReturns(nil, httpsec.Deny, nil)
						authnFilter := Authentication{
							Authenticator: authenticator,
						}
						_, err := authnFilter.Run(req, handler)
						httpErr, ok := err.(*util.HTTPError)
						Expect(ok).To(BeTrue())
						Expect(httpErr.StatusCode).To(Equal(http.StatusUnauthorized))
					})
				})
				Context("Abstain", func() {
					It("should continue with calling handler", func() {
						authenticator.AuthenticateReturns(nil, httpsec.Abstain, nil)
						handler.HandleReturns(nil, errors.New(expectedErrorMessage))
						authnFilter := Authentication{
							Authenticator: authenticator,
						}
						_, err := authnFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						_, isAuthenticated := web.UserFromContext(req.Context())
						Expect(isAuthenticated).To(BeFalse())
					})
				})
				Context("Allow", func() {
					Context("with user", func() {
						It("should add user in request context", func() {
							authenticator.AuthenticateReturns(&web.UserContext{}, httpsec.Allow, nil)
							handler.HandleReturns(nil, errors.New(expectedErrorMessage))
							authnFilter := Authentication{
								Authenticator: authenticator,
							}
							_, err := authnFilter.Run(req, handler)
							checkExpectedErrorMessage(expectedErrorMessage, err)
							user, isAuthenticated := web.UserFromContext(req.Context())
							Expect(isAuthenticated).To(BeTrue())
							Expect(user).ToNot(BeNil())
						})
					})
					Context("without user", func() {
						It("should return error", func() {
							authenticator.AuthenticateReturns(nil, httpsec.Allow, nil)
							handler.HandleReturns(nil, errors.New(expectedErrorMessage))
							authnFilter := Authentication{
								Authenticator: authenticator,
							}
							_, err := authnFilter.Run(req, handler)
							Expect(err).To(Equal(security.ErrUserNotFound))
						})
					})
				})
			})

			Context("when authenticator returns error", func() {
				Context("and decision Abstain", func() {
					It("should return error", func() {
						authenticator.AuthenticateReturns(nil, httpsec.Abstain, errors.New(expectedErrorMessage))
						authnFilter := Authentication{
							Authenticator: authenticator,
						}
						_, err := authnFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
					})
				})

				Context("and decision Deny", func() {
					It("should return http error 403", func() {
						authenticator.AuthenticateReturns(nil, httpsec.Deny, errors.New(expectedErrorMessage))
						authnFilter := Authentication{
							Authenticator: authenticator,
						}
						_, err := authnFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						httpErr, ok := err.(*util.HTTPError)
						Expect(ok).To(BeTrue())
						Expect(httpErr.StatusCode).To(Equal(http.StatusUnauthorized))
					})
				})
			})
		})
	})
})

func checkExpectedErrorMessage(expectedErrorMessage string, err error) {
	Expect(err).To(HaveOccurred())
	Expect(err.Error()).To(ContainSubstring(expectedErrorMessage))
}
