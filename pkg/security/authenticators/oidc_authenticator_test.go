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

package authenticators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/security/http/httpfakes"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sirupsen/logrus"
)

func TestAuthenticators(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OIDC Authenticator")
}

type mockReader struct {
	buff      string
	err       error
	readIndex int64
}

func (e *mockReader) Read(p []byte) (n int, err error) {
	if e.readIndex >= int64(len(e.buff)) {
		err = io.EOF
		return
	}
	n = copy(p, e.buff[e.readIndex:])
	e.readIndex += int64(n)
	return n, e.err
}

type mockReadCloser struct {
	io.Reader
	closeError error
}

func (c *mockReadCloser) Close() error {
	return c.closeError
}

var _ = Describe("OIDC Authenticator", func() {

	ctx := context.TODO()
	var openIDServer *ghttp.Server
	openIDResponseCode := http.StatusOK
	var openIDResponseBody providerJSON
	var openIDResponseBodyBytes []byte

	var readConfigFunc util.DoRequestFunc
	var oauthOptions *OIDCOptions

	issuerPath := "/oauth/token"
	jwksPath := "/public_keys"

	newOpenIDServer := func() *ghttp.Server {
		server := ghttp.NewServer()
		openIDResponseBody = providerJSON{
			Issuer:  server.URL() + issuerPath,
			JWKSURL: server.URL() + jwksPath,
		}
		openIDResponseBodyBytes, _ = json.Marshal(&openIDResponseBody)
		server.RouteToHandler(http.MethodGet, "/.well-known/openid-configuration", func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(openIDResponseCode)
			writer.Write(openIDResponseBodyBytes)
		})
		return server
	}

	BeforeEach(func() {
		openIDServer = newOpenIDServer()
		openIDResponseCode = http.StatusOK
	})

	JustBeforeEach(func() {
		oauthOptions = &OIDCOptions{
			ReadConfigurationFunc: readConfigFunc,
			IssuerURL:             openIDServer.URL(),
			ClientID:              "client-id",
		}
	})

	AfterEach(func() {
		openIDServer.Close()
	})

	Context("NewOIDCAuthenticator", func() {
		Context("When no Issuer URL is present", func() {
			It("Should return an error", func() {
				oauthOptions.IssuerURL = ""
				authenticator, issuer, err := NewOIDCAuthenticator(ctx, oauthOptions)
				Expect(authenticator).To(BeNil())
				Expect(issuer).To(BeEmpty())
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("With custom read config func", func() {
			var body io.ReadCloser
			var readError error

			BeforeEach(func() {
				readConfigFunc = func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: openIDResponseCode,
						Body:       body,
					}, readError
				}
			})

			AfterEach(func() {
				body = ioutil.NopCloser(bytes.NewReader(openIDResponseBodyBytes))
				readError = nil
			})

			Context("When read config returns an error", func() {
				BeforeEach(func() {
					readError = fmt.Errorf("could not read config")
				})
				It("Should return an error", func() {
					_, _, err := NewOIDCAuthenticator(ctx, oauthOptions)
					Expect(err).To(Not(BeNil()))
				})
			})

			Context("When reader returns an error", func() {
				expectedErr := fmt.Errorf("read error")
				BeforeEach(func() {
					openIDResponseCode = http.StatusOK
					body = ioutil.NopCloser(&mockReader{buff: "{}", err: expectedErr})
				})
				It("Should return an error", func() {
					_, _, err := NewOIDCAuthenticator(ctx, oauthOptions)
					Expect(err).To(Not(BeNil()))
					Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
				})
			})

			Context("When response is not a json", func() {
				BeforeEach(func() {
					openIDResponseCode = http.StatusOK
					body = ioutil.NopCloser(&mockReader{buff: "{invalidJson", err: nil})
				})
				It("Should return an error", func() {
					_, _, err := NewOIDCAuthenticator(ctx, oauthOptions)
					Expect(err).To(Not(BeNil()))
				})
			})

			Context("When response body close fails", func() {
				expectedError := fmt.Errorf("Closing failed in mock closer")
				loggingInterceptor := &testutil.LogInterceptor{}
				BeforeEach(func() {
					body = &mockReadCloser{
						Reader:     &mockReader{buff: "{}", err: nil},
						closeError: expectedError,
					}
					logrus.AddHook(loggingInterceptor)
				})

				It("Should log an error", func() {
					NewOIDCAuthenticator(ctx, oauthOptions)
					Expect(loggingInterceptor).To(ContainSubstring(expectedError.Error()))
				})
			})
			Context("When configuration is correct", func() {
				BeforeEach(func() {
					openIDResponseCode = http.StatusOK
				})
				It("Should return an authenticator", func() {
					authenticator, issuer, err := NewOIDCAuthenticator(ctx, oauthOptions)
					Expect(err).To(BeNil())
					Expect(issuer).To(Not(BeEmpty()))
					Expect(authenticator).To(Not(BeNil()))
				})
			})
		})

		Context("With no custom read config func provided", func() {

			BeforeEach(func() {
				readConfigFunc = nil
			})

			Context("When invalid status code is returned", func() {
				BeforeEach(func() {
					openIDResponseCode = http.StatusInternalServerError
				})

				It("Should return error", func() {
					_, _, err := NewOIDCAuthenticator(ctx, oauthOptions)

					Expect(err).To(Not(BeNil()))
				})
			})

			Context("When configuration is correct", func() {
				BeforeEach(func() {
					openIDResponseCode = http.StatusOK
				})

				It("Should return an authenticator", func() {
					authenticator, issuer, err := NewOIDCAuthenticator(ctx, oauthOptions)
					Expect(err).To(BeNil())
					Expect(issuer).To(Not(BeEmpty()))
					Expect(authenticator).To(Not(BeNil()))
				})
			})

			Context("newOIDCConfig", func() {
				It("Should not skip client id check when client id is not empty", func() {
					config := newOIDCConfig(&OIDCOptions{ClientID: "client1"})
					Expect(config.SkipClientIDCheck).To(BeFalse())
				})

				It("Should skip client id check when client id is empty", func() {
					config := newOIDCConfig(&OIDCOptions{ClientID: ""})
					Expect(config.SkipClientIDCheck).To(BeTrue())
				})
			})
		})
	})

	Context("Authenticate", func() {
		var (
			request *http.Request
			err     error
		)
		validateAuthenticationReturns := func(expectedUser *web.UserContext, expectedDecision httpsec.Decision, expectedErr error) {
			authenticator, _, _ := NewOIDCAuthenticator(ctx, oauthOptions)

			user, decision, err := authenticator.Authenticate(&web.Request{Request: request})

			if expectedUser != nil {
				Expect(user).To(Equal(expectedUser))
			} else {
				Expect(user).To(BeNil())
			}

			Expect(decision).To(Equal(expectedDecision))

			if expectedErr != nil {
				Expect(err).To(Equal(expectedErr))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		}

		BeforeEach(func() {
			request, err = http.NewRequest(http.MethodGet, "https://example.com", &mockReader{err: nil, buff: ""})
			Expect(err).ShouldNot(HaveOccurred())

			readConfigFunc = func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: openIDResponseCode,
					Body:       ioutil.NopCloser(bytes.NewReader(openIDResponseBodyBytes)),
				}, nil
			}
		})

		Context("when Authorization header is missing", func() {
			It("should abstain authentication decision with no error", func() {
				Expect(request.Header.Get("Authorization")).To(Equal(""))

				validateAuthenticationReturns(nil, httpsec.Abstain, nil)
			})
		})

		Context("when Authorization header is empty", func() {
			It("should abstain authentication decision with no error", func() {
				request.Header.Set("Authorization", "")

				validateAuthenticationReturns(nil, httpsec.Abstain, nil)
			})
		})

		Context("when Authorization header is not bearer", func() {
			It("should abstain authentication decision with no error", func() {
				request.Header.Set("Authorization", "Basic admin:admin")

				validateAuthenticationReturns(nil, httpsec.Abstain, nil)
			})
		})

		Context("when Authorization header is bearer", func() {
			Context("when token is missing", func() {
				It("should deny authentication with no error", func() {
					request.Header.Set("Authorization", "bearer ")

					validateAuthenticationReturns(nil, httpsec.Deny, nil)
				})
			})

			Context("when token is present", func() {
				var (
					verifier      *httpfakes.FakeTokenVerifier
					authenticator httpsec.Authenticator
					expectedError error
				)

				verifyTokenCases := func() {
					Context("when verifier returns an error", func() {
						BeforeEach(func() {
							expectedError = fmt.Errorf("Verifier returned error")

							verifier.VerifyReturns(nil, expectedError)
						})

						It("should deny with an error", func() {
							user, decision, err := authenticator.Authenticate(&web.Request{Request: request})

							Expect(user).To(BeNil())
							Expect(decision).To(Equal(httpsec.Deny))
							Expect(err).To(Equal(expectedError))
						})
					})

					Context("when returned token cannot extract claims", func() {
						var fakeToken *httpfakes.FakeTokenData

						BeforeEach(func() {
							expectedError = fmt.Errorf("Claims extraction error")

							fakeToken = &httpfakes.FakeTokenData{}
							fakeToken.ClaimsReturns(expectedError)

							verifier.VerifyReturns(fakeToken, nil)

						})

						It("should deny with an error", func() {
							user, decision, err := authenticator.Authenticate(&web.Request{Request: request})

							Expect(user).To(BeNil())
							Expect(decision).To(Equal(httpsec.Deny))
							Expect(err).To(Equal(expectedError))
						})
					})

					Context("when returned token is valid", func() {
						expectedUserName := "test_user"

						BeforeEach(func() {
							tokenJSON := fmt.Sprintf(`{"user_name": "%s", "abc": "xyz"}`, expectedUserName)
							token := &httpfakes.FakeTokenData{}
							token.ClaimsStub = func(v interface{}) error {
								return json.Unmarshal([]byte(tokenJSON), v)
							}
							verifier.VerifyReturns(token, nil)
						})

						It("should allow authentication and return user", func() {
							user, decision, err := authenticator.Authenticate(&web.Request{Request: request})

							Expect(user).To(Not(BeNil()))
							Expect(user.Name).To(Equal(expectedUserName))
							Expect(decision).To(Equal(httpsec.Allow))
							Expect(err).To(BeNil())

							claims := struct {
								Abc string
							}{}

							err = user.Data(&claims)

							_, token := verifier.VerifyArgsForCall(0)
							Expect(token).To(Equal("token"))
							Expect(err).To(BeNil())
							Expect(claims.Abc).To(Equal("xyz"))
						})
					})
				}

				Context("when Bearer starts with uppercase", func() {
					BeforeEach(func() {
						verifier = &httpfakes.FakeTokenVerifier{}
						authenticator = &OauthAuthenticator{Verifier: verifier}

						request.Header.Set("Authorization", "Bearer token")
					})

					verifyTokenCases()
				})

				Context("when bearer starts with lowercase", func() {
					BeforeEach(func() {
						verifier = &httpfakes.FakeTokenVerifier{}
						authenticator = &OauthAuthenticator{Verifier: verifier}

						request.Header.Set("Authorization", "bearer token")
					})

					verifyTokenCases()
				})
			})
		})
	})
})
