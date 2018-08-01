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

package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"errors"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/security/securityfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sirupsen/logrus"
)

func TestApi(t *testing.T) {
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

type loggingInterceptorHook struct {
	data []byte
}

func (*loggingInterceptorHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *loggingInterceptorHook) Fire(entry *logrus.Entry) error {
	str, _ := entry.String()
	hook.data = append(hook.data, []byte(str)...)
	return nil
}

var _ = Describe("OIDC Authenticator", func() {

	ctx := context.TODO()
	var openidServer *ghttp.Server
	openIdResponseCode := http.StatusOK
	var openIdResponseBody providerJSON
	var openIdResponseBodyBytes []byte

	var readConfigFunc util.DoRequestFunc
	var options Options

	issuerPath := "/oauth/token"
	jwksPath := "/public_keys"

	newOpenIdServer := func() *ghttp.Server {
		server := ghttp.NewServer()
		openIdResponseBody = providerJSON{
			Issuer:  server.URL() + issuerPath,
			JWKSURL: server.URL() + jwksPath,
		}
		openIdResponseBodyBytes, _ = json.Marshal(&openIdResponseBody)
		server.RouteToHandler(http.MethodGet, "/.well-known/openid-configuration", func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(openIdResponseCode)
			writer.Write(openIdResponseBodyBytes)
		})
		return server
	}

	BeforeEach(func() {
		openidServer = newOpenIdServer()
		openIdResponseCode = http.StatusOK
	})

	JustBeforeEach(func() {
		options = Options{
			ReadConfigurationFunc: readConfigFunc,
			IssuerURL:             openidServer.URL(),
			ClientID:              "client-id",
		}
	})

	AfterEach(func() {
		openidServer.Close()
	})

	Context("NewAuthenticator", func() {
		Context("When no Issuer URL is present", func() {
			It("Should return an error", func() {
				options.IssuerURL = ""
				authenticator, err := NewAuthenticator(ctx, options)
				Expect(authenticator).To(BeNil())
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("With custom read config func", func() {
			var body io.ReadCloser
			var readError error

			BeforeEach(func() {
				readConfigFunc = func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: openIdResponseCode,
						Body:       body,
					}, readError
				}
			})

			AfterEach(func() {
				body = ioutil.NopCloser(bytes.NewReader(openIdResponseBodyBytes))
				readError = nil
			})

			Context("When read config returns an error", func() {
				BeforeEach(func() {
					readError = fmt.Errorf("could not read config")
				})
				It("Should return an error", func() {
					_, err := NewAuthenticator(ctx, options)
					Expect(err).To(Not(BeNil()))
				})
			})

			Context("When reader returns an error", func() {
				expectedErr := fmt.Errorf("read error")
				BeforeEach(func() {
					openIdResponseCode = http.StatusOK
					body = ioutil.NopCloser(&mockReader{buff: "{}", err: expectedErr})
				})
				It("Should return an error", func() {
					_, err := NewAuthenticator(ctx, options)
					Expect(err).To(Not(BeNil()))
					Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
				})
			})

			Context("When response is not a json", func() {
				BeforeEach(func() {
					openIdResponseCode = http.StatusOK
					body = ioutil.NopCloser(&mockReader{buff: "{invalidJson", err: nil})
				})
				It("Should return an error", func() {
					_, err := NewAuthenticator(ctx, options)
					Expect(err).To(Not(BeNil()))
				})
			})

			Context("When response body close fails", func() {
				expectedError := fmt.Errorf("Closing failed in mock closer")
				loggingInterceptor := &loggingInterceptorHook{}
				BeforeEach(func() {
					body = &mockReadCloser{
						Reader:     &mockReader{buff: "{}", err: nil},
						closeError: expectedError,
					}
					logrus.AddHook(loggingInterceptor)
				})

				It("Should log an error", func() {
					NewAuthenticator(ctx, options)
					loggedError := string(loggingInterceptor.data)
					Expect(loggedError).To(ContainSubstring(expectedError.Error()))
				})
			})
			Context("When configuration is correct", func() {
				BeforeEach(func() {
					openIdResponseCode = http.StatusOK
				})
				It("Should return an authenticator", func() {
					authenticator, err := NewAuthenticator(ctx, options)
					Expect(err).To(BeNil())
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
					openIdResponseCode = http.StatusInternalServerError
				})

				It("Should return error", func() {
					_, err := NewAuthenticator(ctx, options)

					Expect(err).To(Not(BeNil()))
				})
			})

			Context("When configuration is correct", func() {

				BeforeEach(func() {
					openIdResponseCode = http.StatusOK
				})

				It("Should return an authenticator", func() {
					authenticator, err := NewAuthenticator(ctx, options)
					Expect(err).To(BeNil())
					Expect(authenticator).To(Not(BeNil()))
				})
			})
		})
	})

	Context("Authenticate", func() {
		var (
			request *http.Request
			err     error
		)
		validateAuthenticationReturns := func(expectedUser *security.User, expectedDecision security.AuthenticationDecision, expectedErr error) {
			authenticator, _ := NewAuthenticator(ctx, options)

			user, decision, err := authenticator.Authenticate(request)

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
					StatusCode: openIdResponseCode,
					Body:       ioutil.NopCloser(bytes.NewReader(openIdResponseBodyBytes)),
				}, nil
			}
		})

		Context("when Authorization header is missing", func() {
			It("should abstain authentication decision with no error", func() {
				Expect(request.Header.Get("Authorization")).To(Equal(""))

				validateAuthenticationReturns(nil, security.Abstain, nil)
			})
		})

		Context("when Authorization header is empty", func() {
			It("should abstain authentication decision with no error", func() {
				request.Header.Set("Authorization", "")

				validateAuthenticationReturns(nil, security.Abstain, nil)
			})
		})

		Context("When Authorization header is not bearer", func() {
			It("should abstain authentication decision with no error", func() {
				request.Header.Set("Authorization", "Basic admin:admin")

				validateAuthenticationReturns(nil, security.Abstain, nil)
			})
		})

		Context("when Authorization header is bearer", func() {
			Context("when token is missing", func() {
				It("should deny authentication with malformed token error", func() {
					request.Header.Set("Authorization", "bearer ")

					validateAuthenticationReturns(nil, security.Deny, errors.New("oidc: malformed jwt: square/go-jose: compact JWS format must have three parts"))
				})
			})
			Context("when token is present", func() {
				var (
					verifier      *securityfakes.FakeTokenVerifier
					authenticator security.Authenticator
					expectedError error
				)

				BeforeEach(func() {
					verifier = &securityfakes.FakeTokenVerifier{}
					authenticator = &Authenticator{Verifier: verifier}

					request.Header.Set("Authorization", "Bearer token")
				})

				Context("when verifier returns an error", func() {
					BeforeEach(func() {
						expectedError = fmt.Errorf("Verifier returned error")

						verifier.VerifyReturns(nil, expectedError)
					})

					It("should deby with an error", func() {
						user, decision, err := authenticator.Authenticate(request)

						Expect(user).To(BeNil())
						Expect(decision).To(Equal(security.Deny))
						Expect(err).To(Equal(expectedError))
					})
				})

				Context("when returned token cannot extract claims", func() {
					var fakeToken *securityfakes.FakeToken

					BeforeEach(func() {
						expectedError = fmt.Errorf("Claims extraction error")

						fakeToken = &securityfakes.FakeToken{}
						fakeToken.ClaimsReturns(expectedError)

						verifier.VerifyReturns(fakeToken, nil)

					})

					It("should deny with an error", func() {
						user, decision, err := authenticator.Authenticate(request)

						Expect(user).To(BeNil())
						Expect(decision).To(Equal(security.Deny))
						Expect(err).To(Equal(expectedError))
					})
				})

				Context("when returned token is valid", func() {
					expectedUserName := "test_user"

					BeforeEach(func() {
						tokenJson := fmt.Sprintf("{\"user_name\": \"%s\"}", expectedUserName)
						token := &securityfakes.FakeToken{}
						token.ClaimsStub = func(v interface{}) error {
							return json.Unmarshal([]byte(tokenJson), v)
						}
						verifier.VerifyReturns(token, nil)
					})

					It("should allow authentication and return user", func() {
						user, decision, err := authenticator.Authenticate(request)

						Expect(user).To(Not(BeNil()))
						Expect(user.Name).To(Equal(expectedUserName))
						Expect(decision).To(Equal(security.Allow))
						Expect(err).To(BeNil())
					})
				})
			})
		})
	})
})
