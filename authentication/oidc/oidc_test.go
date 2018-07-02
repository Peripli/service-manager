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

	"github.com/Peripli/service-manager/authentication"
	"github.com/Peripli/service-manager/authentication/authenticationfakes"
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

var _ = Describe("OIDC", func() {

	ctx := context.TODO()
	var openidServer *ghttp.Server
	openIdResponseCode := http.StatusOK
	var openIdResponseBody providerJSON
	var openIdResponseBodyBytes []byte

	var readConfigFunc DoRequestFunc
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

	Context("New Authenticator", func() {

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
					Expect(err.Error()).To(ContainSubstring("Unexpected status code"))
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

		request, _ := http.NewRequest(http.MethodGet, "https://example.com", &mockReader{err: nil, buff: ""})
		validateAuthenticateReturnsError := func() {
			authenticator, _ := NewAuthenticator(ctx, options)
			user, err := authenticator.Authenticate(request)
			Expect(user).To(BeNil())
			Expect(err).To(Not(BeNil()))
		}

		BeforeEach(func() {
			readConfigFunc = func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: openIdResponseCode,
					Body:       ioutil.NopCloser(bytes.NewReader(openIdResponseBodyBytes)),
				}, nil
			}
		})

		Context("When Authorization header is missing", func() {
			It("Should return error", func() {
				validateAuthenticateReturnsError()
			})
		})
		Context("When Authorization header is empty", func() {
			It("Should return error", func() {
				request.Header.Set("Authorization", "")
				validateAuthenticateReturnsError()
			})
		})
		Context("When Authorization header is not bearer", func() {
			It("Should return an error", func() {
				request.Header.Set("Authorization", "Basic admin:admin")
				validateAuthenticateReturnsError()
			})
		})

		Context("When Bearer header has empty token", func() {
			It("Should return an error", func() {
				request.Header.Set("Authorization", "bearer ")
				validateAuthenticateReturnsError()
			})
		})

		Context("When Bearer has token", func() {

			var verifier = &authenticationfakes.FakeTokenVerifier{}
			var authenticator authentication.Authenticator
			var expectedError error

			BeforeEach(func() {
				request.Header.Set("Authorization", "bearer token")
				authenticator = &Authenticator{Verifier: verifier}
			})

			Context("When verifier returns an error", func() {
				BeforeEach(func() {
					expectedError = fmt.Errorf("Verifier returned error")

					verifier.VerifyReturns(nil, expectedError)
				})

				It("Should return an error", func() {
					user, err := authenticator.Authenticate(request)
					Expect(user).To(BeNil())
					Expect(err).To(Equal(expectedError))
				})
			})

			Context("When returned token cannot extract claims", func() {
				BeforeEach(func() {
					expectedError = fmt.Errorf("Claims extraction error")

					fakeToken := &authenticationfakes.FakeToken{}
					fakeToken.ClaimsReturns(expectedError)
					verifier.VerifyReturns(fakeToken, nil)

				})
				It("Should return error", func() {
					user, err := authenticator.Authenticate(request)
					Expect(user).To(BeNil())
					Expect(err).To(Equal(expectedError))
				})
			})

			Context("When returned token is valid", func() {
				expectedUserName := "test_user"

				BeforeEach(func() {
					tokenJson := fmt.Sprintf("{\"user_name\": \"%s\"}", expectedUserName)
					token := &authenticationfakes.FakeToken{}
					token.ClaimsStub = func(v interface{}) error {
						return json.Unmarshal([]byte(tokenJson), v)
					}
					verifier.VerifyReturns(token, nil)
				})

				It("Should return user", func() {
					authenticator := &Authenticator{Verifier: verifier}
					user, err := authenticator.Authenticate(request)
					Expect(user).To(Not(BeNil()))
					Expect(user.Name).To(Equal(expectedUserName))
					Expect(err).To(BeNil())
				})
			})
		})
	})
})
