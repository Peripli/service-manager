/*
 * Copyright 2018 The Service Manager Authors
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

package authn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/security/securityfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authn Suite")
}

type testStructure struct {
	authnResp                *security.User
	authnDecision            security.AuthenticationDecision
	authnErr                 error
	request                  *web.Request
	actualHandlerInvokations int
	expectedErr              error
	expectedResp             *web.Response
}

var _ = Describe("Authn", func() {
	var (
		fakeAuthenticator   *securityfakes.FakeAuthenticator
		fakeHandler         *webfakes.FakeHandler
		expectedWebResponse *web.Response
		authnError          error
	)

	BeforeEach(func() {
		fakeAuthenticator = &securityfakes.FakeAuthenticator{}
		fakeHandler = &webfakes.FakeHandler{}
		expectedWebResponse = &web.Response{
			StatusCode: http.StatusOK,
			Body:       []byte(`{}`),
		}

		authnError = errors.New("error")

	})

	validateMiddlewareBehavesCorrectly := func(middleware web.Middleware, t *testStructure) {
		fakeAuthenticator.AuthenticateReturns(t.authnResp, t.authnDecision, t.authnErr)
		fakeHandler.HandleReturns(expectedWebResponse, nil)
		resp, err := middleware.Run(fakeHandler).Handle(t.request)

		Expect(fakeHandler.HandleCallCount()).To(Equal(t.actualHandlerInvokations))

		if t.expectedResp != nil {
			Expect(resp).To(Equal(t.expectedResp))
		} else {
			Expect(resp).To(BeNil())
		}

		if t.expectedErr != nil {
			Expect(err).To(Equal(t.expectedErr))

		} else {
			Expect(err).To(BeNil())
		}
	}

	errUnauthorizedWithDescription := func(description string) error {
		return &util.HTTPError{
			ErrorType:   "Unauthorized",
			Description: description,
			StatusCode:  http.StatusUnauthorized,
		}
	}

	reqWithContext := func(ctx context.Context) *web.Request {
		httpReq := httptest.NewRequest("GET", "/", strings.NewReader("")).WithContext(ctx)
		webReq := &web.Request{Request: httpReq}
		return webReq
	}

	Describe("Authentication Middleware", func() {
		validateAuthnMiddlewareBehavesCorrectly := func(t *testStructure) {
			validateMiddlewareBehavesCorrectly(&Middleware{
				authenticator: fakeAuthenticator,
				name:          "name",
			}, t)
		}

		Context("when user is already authenticated and present in context", func() {
			It("invokes the next handler", func() {
				validateAuthnMiddlewareBehavesCorrectly(&testStructure{
					request:                  reqWithContext(context.WithValue(context.Background(), userKey, &security.User{Name: "username"})),
					actualHandlerInvokations: 1,
					expectedResp:             expectedWebResponse,
				})
			})

		})

		Context("when authenticator abstains from taking authentication decision", func() {
			Context("and returns an error", func() {
				It("propagates the same error", func() {
					validateAuthnMiddlewareBehavesCorrectly(&testStructure{
						authnErr:                 authnError,
						authnDecision:            security.Abstain,
						request:                  reqWithContext(context.Background()),
						actualHandlerInvokations: 0,
						expectedErr:              authnError,
					})
				})
			})

			Context("and returns no error", func() {
				It("invokes the next handler", func() {
					validateAuthnMiddlewareBehavesCorrectly(&testStructure{
						authnErr:                 nil,
						authnDecision:            security.Abstain,
						request:                  reqWithContext(context.Background()),
						actualHandlerInvokations: 1,
						expectedResp:             expectedWebResponse,
					})
				})
			})
		})

		Context("when authenticator allows authentication", func() {
			Context("and returns an error", func() {
				It("propagates the same error", func() {
					validateAuthnMiddlewareBehavesCorrectly(&testStructure{
						authnDecision:            security.Allow,
						authnErr:                 authnError,
						request:                  reqWithContext(context.Background()),
						actualHandlerInvokations: 0,
						expectedErr:              authnError,
					})
				})
			})

			Context("and returns no error", func() {
				Context("and returns no user", func() {
					It("returns a missing user info error", func() {
						validateAuthnMiddlewareBehavesCorrectly(&testStructure{
							authnErr:                 nil,
							authnResp:                nil,
							authnDecision:            security.Allow,
							request:                  reqWithContext(context.Background()),
							actualHandlerInvokations: 0,
							expectedErr:              errUserNotFound,
						})
					})
				})

				Context("and returns a user", func() {
					It("invokes the next handler and adds the user to the request context", func() {
						testStruct := &testStructure{
							authnResp: &security.User{
								Name: "username",
							},
							authnDecision:            security.Allow,
							authnErr:                 nil,
							request:                  reqWithContext(context.Background()),
							actualHandlerInvokations: 1,
							expectedErr:              nil,
							expectedResp:             expectedWebResponse,
						}

						validateAuthnMiddlewareBehavesCorrectly(testStruct)

						_, ok := UserFromContext(testStruct.request.Context())
						Expect(ok).To(BeTrue())
					})
				})
			})
		})

		Context("when authenticator denies authentication", func() {
			Context("and returns no error", func() {
				It("returns a generic authorization denied HTTPError", func() {
					validateAuthnMiddlewareBehavesCorrectly(&testStructure{
						authnDecision:            security.Deny,
						authnErr:                 nil,
						request:                  reqWithContext(context.Background()),
						expectedErr:              errUnauthorized,
						actualHandlerInvokations: 0,
					})
				})
			})

			Context("and returns an error", func() {
				It("returns an HTTPError containg the authentication error", func() {
					validateAuthnMiddlewareBehavesCorrectly(&testStructure{
						authnDecision:            security.Deny,
						authnErr:                 authnError,
						request:                  reqWithContext(context.Background()),
						actualHandlerInvokations: 0,

						expectedErr: errUnauthorizedWithDescription(authnError.Error()),
					})
				})
			})
		})
	})

	Describe("Authentication Required Middleware", func() {
		var (
			fakeHandler         *webfakes.FakeHandler
			authnRequiredFilter *RequiredAuthnFilter
		)

		validateRequiredMiddlewareBehavesCorrectly := func(t *testStructure) {
			validateMiddlewareBehavesCorrectly(web.MiddlewareFunc(authnRequiredFilter.Run), t)
		}

		BeforeEach(func() {
			fakeHandler = &webfakes.FakeHandler{}
			authnRequiredFilter = NewRequiredAuthnFilter()
		})

		Context("when user is in context", func() {
			It("invokes next handler", func() {
				validateRequiredMiddlewareBehavesCorrectly(&testStructure{
					request:                  reqWithContext(context.WithValue(context.Background(), userKey, &security.User{Name: "username"})),
					actualHandlerInvokations: 1,
					expectedErr:              nil,
					expectedResp:             expectedWebResponse,
				})
			})
		})

		Context("when user is missing from context", func() {
			It("returns unauthorized error", func() {
				validateRequiredMiddlewareBehavesCorrectly(&testStructure{
					request:                  reqWithContext(context.Background()),
					actualHandlerInvokations: 0,
					expectedErr:              errUnauthorized,
					expectedResp:             nil,
				})
			})
		})
	})
})
