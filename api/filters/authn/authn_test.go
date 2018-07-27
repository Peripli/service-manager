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

var _ = Describe("Authn", func() {

	Describe("Middleware", func() {
		Describe("Run", func() {
			Context("With authentication propagated to next handler", func() {
				It("Should invoke next handler", func() {
					fakeAuthenticator := getFakeAuthenticator(nil, false, nil)
					fakeHandler := getFakeHandler(nil, errors.New("expected"))
					middleware := getMiddleware(fakeAuthenticator, "name")
					fakeRequest := getWebRequest(httptest.NewRequest("GET", "/", strings.NewReader("")))
					_, err := middleware.Run(fakeHandler).Handle(fakeRequest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("expected"))
				})
			})

			Context("With authentication already successfull", func() {
				It("Should invoke next handler", func() {
					fakeAuthenticator := getFakeAuthenticator(nil, true, nil)
					fakeHandler := getFakeHandler(nil, errors.New("expected"))
					middleware := getMiddleware(fakeAuthenticator, "name")
					customContext := context.WithValue(context.Background(), UserKey, &security.User{Name: "username"})
					httpRequest := httptest.NewRequest("GET", "/", strings.NewReader(""))
					fakeRequest := getWebRequest(httpRequest.WithContext(customContext))
					_, err := middleware.Run(fakeHandler).Handle(fakeRequest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("expected"))
				})
			})

			Context("With authentication returning error", func() {
				It("Should return the error", func() {
					fakeAuthenticator := getFakeAuthenticator(nil, true, errors.New("expected"))
					fakeHandler := getFakeHandler(nil, errors.New("must not happen"))
					middleware := getMiddleware(fakeAuthenticator, "name")
					fakeRequest := getWebRequest(httptest.NewRequest("GET", "/", strings.NewReader("")))
					_, err := middleware.Run(fakeHandler).Handle(fakeRequest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("authentication failed"))
				})
			})

			Context("With authentication returning nil user", func() {
				It("Should return the error", func() {
					fakeAuthenticator := getFakeAuthenticator(nil, true, nil)
					fakeHandler := getFakeHandler(nil, errors.New("must not happen"))
					middleware := getMiddleware(fakeAuthenticator, "name")
					fakeRequest := getWebRequest(httptest.NewRequest("GET", "/", strings.NewReader("")))
					_, err := middleware.Run(fakeHandler).Handle(fakeRequest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("authentication failed: username identity could not be established"))
				})
			})

			Context("With authentication success on current authenticator", func() {
				It("Should invoke next handler", func() {
					fakeAuthenticator := getFakeAuthenticator(&security.User{Name: "username"}, true, nil)
					fakeHandler := getFakeHandler(nil, errors.New("expected"))
					middleware := getMiddleware(fakeAuthenticator, "name")
					fakeRequest := getWebRequest(httptest.NewRequest("GET", "/", strings.NewReader("")))
					_, err := middleware.Run(fakeHandler).Handle(fakeRequest)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("expected"))
				})
			})
		})
	})

})

func getFakeAuthenticator(user *security.User, used bool, err error) *securityfakes.FakeAuthenticator {
	fakeAuthenticator := &securityfakes.FakeAuthenticator{}
	fakeAuthenticator.AuthenticateReturns(user, used, err)
	return fakeAuthenticator
}

func getFakeHandler(resp *web.Response, err error) *webfakes.FakeHandler {
	fakeHandler := &webfakes.FakeHandler{}
	fakeHandler.HandleReturns(resp, err)
	return fakeHandler
}

func getMiddleware(authenticator security.Authenticator, name string) *Middleware {
	return &Middleware{
		authenticator: authenticator,
		name:          name,
	}
}

func getWebRequest(req *http.Request) *web.Request {
	return &web.Request{Request: req}
}
