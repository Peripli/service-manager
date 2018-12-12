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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/security/securityfakes"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Security Middlewares Suite")
}

var _ = Describe("Middlewares tests", func() {
	const expectedErrorMessage = "expected error"
	var (
		req     *web.Request
		handler *webfakes.FakeHandler
	)

	Describe("Authz middleware", func() {
		const filterName = "authzFilterName"
		var authorizer *securityfakes.FakeAuthorizer

		BeforeEach(func() {
			req = &web.Request{
				Request: httptest.NewRequest("GET", "/", nil),
			}
			authorizer = &securityfakes.FakeAuthorizer{}
			handler = &webfakes.FakeHandler{}
		})

		Describe("when Filter.FilterMatchers is invoked", func() {
			It("should panic", func() {
				authzFilter := NewAuthzMiddleware(filterName, nil)
				Expect(func() {
					authzFilter.FilterMatchers()
				}).To(Panic())
			})
		})

		Describe("when Filter.Name is invoked", func() {
			It("should return the name", func() {
				authzFilter := NewAuthzMiddleware(filterName, nil)
				Expect(authzFilter.Name()).To(Equal(filterName))
			})
		})

		Describe("when Filter.Run is invoked", func() {

			Context("when authorizer returns decision", func() {
				Context("Deny", func() {
					It("should return error", func() {
						authorizer.AuthorizeReturns(security.Deny, nil)
						authzFilter := NewAuthzMiddleware(filterName, authorizer)
						_, err := authzFilter.Run(req, handler)
						httpErr, ok := err.(*util.HTTPError)
						Expect(ok).To(BeTrue())
						Expect(httpErr.StatusCode).To(Equal(http.StatusForbidden))
					})
				})
				Context("Abstain", func() {
					It("should continue with calling handler", func() {
						authorizer.AuthorizeReturns(security.Abstain, nil)
						handler.HandleReturns(nil, errors.New(expectedErrorMessage))
						authzFilter := NewAuthzMiddleware(filterName, authorizer)
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						Expect(web.IsAuthorized(req.Context())).To(BeFalse())
					})
				})
				Context("Allow", func() {
					It("should add authorization flag in request context", func() {
						authorizer.AuthorizeReturns(security.Allow, nil)
						handler.HandleReturns(nil, errors.New(expectedErrorMessage))
						authzFilter := NewAuthzMiddleware(filterName, authorizer)
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						Expect(web.IsAuthorized(req.Context())).To(BeTrue())
					})
				})
			})

			Context("when authorizer returns error", func() {
				Context("and decision Abstain", func() {
					It("should return error", func() {
						authorizer.AuthorizeReturns(security.Abstain, errors.New(expectedErrorMessage))
						authzFilter := NewAuthzMiddleware(filterName, authorizer)
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
					})
				})

				Context("and decision Deny", func() {
					It("should return http error 403", func() {
						authorizer.AuthorizeReturns(security.Deny, errors.New(expectedErrorMessage))
						authzFilter := NewAuthzMiddleware(filterName, authorizer)
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
		const filterName = "authnFilterName"

		var authenticator *securityfakes.FakeAuthenticator

		BeforeEach(func() {
			req = &web.Request{
				Request: httptest.NewRequest("GET", "/", nil),
			}
			authenticator = &securityfakes.FakeAuthenticator{}
			handler = &webfakes.FakeHandler{}
		})

		Describe("when Filter.FilterMatchers is invoked", func() {
			It("should panic", func() {
				authnFilter := NewAuthnMiddleware(filterName, nil)
				Expect(func() {
					authnFilter.FilterMatchers()
				}).To(Panic())
			})
		})

		Describe("when Filter.Name is invoked", func() {
			It("should return the name", func() {
				authnFilter := NewAuthnMiddleware(filterName, nil)
				Expect(authnFilter.Name()).To(Equal(filterName))
			})
		})

		Describe("when Filter.Run is invoked", func() {

			Context("when authentication already passed", func() {
				It("should continue", func() {
					authnFilter := NewAuthnMiddleware(filterName, nil)
					req.Request = req.Request.WithContext(web.ContextWithUser(req.Context(), &web.UserContext{}))
					authnFilter.Run(req, handler)
					Expect(handler.HandleCallCount()).To(Equal(1))
				})
			})

			Context("when authenticator returns decision", func() {
				Context("Deny", func() {
					It("should return error", func() {
						authenticator.AuthenticateReturns(nil, security.Deny, nil)
						authzFilter := NewAuthnMiddleware(filterName, authenticator)
						_, err := authzFilter.Run(req, handler)
						httpErr, ok := err.(*util.HTTPError)
						Expect(ok).To(BeTrue())
						Expect(httpErr.StatusCode).To(Equal(http.StatusUnauthorized))
					})
				})
				Context("Abstain", func() {
					It("should continue with calling handler", func() {
						authenticator.AuthenticateReturns(nil, security.Abstain, nil)
						handler.HandleReturns(nil, errors.New(expectedErrorMessage))
						authzFilter := NewAuthnMiddleware(filterName, authenticator)
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
						_, isAuthenticated := web.UserFromContext(req.Context())
						Expect(isAuthenticated).To(BeFalse())
					})
				})
				Context("Allow", func() {
					Context("with user", func() {
						It("should add user in request context", func() {
							authenticator.AuthenticateReturns(&web.UserContext{}, security.Allow, nil)
							handler.HandleReturns(nil, errors.New(expectedErrorMessage))
							authzFilter := NewAuthnMiddleware(filterName, authenticator)
							_, err := authzFilter.Run(req, handler)
							checkExpectedErrorMessage(expectedErrorMessage, err)
							user, isAuthenticated := web.UserFromContext(req.Context())
							Expect(isAuthenticated).To(BeTrue())
							Expect(user).ToNot(BeNil())
						})
					})
					Context("without user", func() {
						It("should return error", func() {
							authenticator.AuthenticateReturns(nil, security.Allow, nil)
							handler.HandleReturns(nil, errors.New(expectedErrorMessage))
							authzFilter := NewAuthnMiddleware(filterName, authenticator)
							_, err := authzFilter.Run(req, handler)
							Expect(err).To(Equal(security.ErrUserNotFound))
						})
					})
				})
			})

			Context("when authenticator returns error", func() {
				Context("and decision Abstain", func() {
					It("should return error", func() {
						authenticator.AuthenticateReturns(nil, security.Abstain, errors.New(expectedErrorMessage))
						authzFilter := NewAuthnMiddleware(filterName, authenticator)
						_, err := authzFilter.Run(req, handler)
						checkExpectedErrorMessage(expectedErrorMessage, err)
					})
				})

				Context("and decision Deny", func() {
					It("should return http error 403", func() {
						authenticator.AuthenticateReturns(nil, security.Deny, errors.New(expectedErrorMessage))
						authzFilter := NewAuthnMiddleware(filterName, authenticator)
						_, err := authzFilter.Run(req, handler)
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
	Expect(err.Error()).To(Equal(expectedErrorMessage))
}
