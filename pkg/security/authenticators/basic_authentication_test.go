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

package authenticators_test

import (
	"encoding/base64"
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/api/osb"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/storagefakes"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/authenticators"
	httpsec "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

var _ = Describe("Basic Authenticator", func() {
	var user string
	var password string
	var basicHeader string
	var request *http.Request
	var err error

	var fakeRepository *storagefakes.FakeStorage

	var authenticator *authenticators.Basic

	BeforeEach(func() {
		user = "username"
		password = "password"
		basicHeader = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))

		fakeRepository = &storagefakes.FakeStorage{}

		authenticator = &authenticators.Basic{
			Repository: fakeRepository,
		}

		request, err = http.NewRequest(http.MethodGet, "https://example.com/v1/osb/123/v2/catalog", nil)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("Authenticate", func() {
		Context("When authorization is not basic", func() {
			It("Should abstain", func() {
				request.Header.Add("Authorization", "Bearer token")
				user, decision, err := authenticator.Authenticate(&web.Request{Request: request})
				Expect(err).ToNot(HaveOccurred())
				Expect(user).To(BeNil())
				Expect(decision).To(Equal(httpsec.Abstain))
			})
		})

		Context("when authorization is basic", func() {
			BeforeEach(func() {
				request.Header.Add("Authorization", "Basic "+basicHeader)
			})

			Context("platform credentials", func() {
				BeforeEach(func() {
					authenticator.BasicAuthenticatorFunc = authenticators.BasicPlatformAuthenticator
				})

				Context("When no platforms are found", func() {
					BeforeEach(func() {
						fakeRepository.ListReturns(&types.Platforms{}, nil)
					})

					It("Should deny", func() {
						user, decision, err := authenticator.Authenticate(&web.Request{Request: request})
						Expect(err).To(HaveOccurred())
						Expect(user).To(BeNil())
						Expect(decision).To(Equal(httpsec.Deny))
					})
				})

				Context("when more than one platform is found", func() {
					BeforeEach(func() {
						fakeRepository.ListReturns(&types.Platforms{
							Platforms: []*types.Platform{
								{
									Base: types.Base{
										ID: "id1",
									},
									Credentials: &types.Credentials{
										Basic: &types.Basic{
											Username: "username",
											Password: "password",
										},
									},
								},
								{
									Base: types.Base{
										ID: "id2",
									},
									Credentials: &types.Credentials{
										Basic: &types.Basic{
											Username: "username",
											Password: "password2",
										},
									},
								},
							},
						}, nil)
					})

					It("Should deny", func() {
						user, decision, err := authenticator.Authenticate(&web.Request{Request: request})
						Expect(err).To(HaveOccurred())
						Expect(user).To(BeNil())
						Expect(decision).To(Equal(httpsec.Deny))
					})
				})

				Context("When getting platforms from storage fails", func() {
					expectedError := fmt.Errorf("error")

					BeforeEach(func() {
						fakeRepository.ListReturns(nil, expectedError)
					})

					It("Should abstain with error", func() {
						user, decision, err := authenticator.Authenticate(&web.Request{Request: request})
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
						Expect(user).To(BeNil())
						Expect(decision).To(Equal(httpsec.Abstain))
					})
				})

				Context("When passwords do not match", func() {
					BeforeEach(func() {
						fakeRepository.ListReturns(&types.Platforms{
							Platforms: []*types.Platform{
								{
									Base: types.Base{
										ID: "id1",
									},
									Credentials: &types.Credentials{
										Basic: &types.Basic{
											Username: "username",
											Password: "not-matching-password",
										},
									},
								},
							},
						}, nil)
					})

					It("Should deny", func() {
						user, decision, err := authenticator.Authenticate(&web.Request{Request: request})
						Expect(err).To(HaveOccurred())
						Expect(user).To(BeNil())
						Expect(decision).To(Equal(httpsec.Deny))
					})
				})

				Context("When passwords match", func() {
					BeforeEach(func() {
						fakeRepository.ListReturns(&types.Platforms{
							Platforms: []*types.Platform{
								{
									Base: types.Base{
										ID: "id1",
									},
									Credentials: &types.Credentials{
										Basic: &types.Basic{
											Username: "username",
											Password: "password",
										},
									},
								},
							},
						}, nil)
					})

					It("Should allow", func() {
						user, decision, err := authenticator.Authenticate(&web.Request{Request: request})
						Expect(err).ToNot(HaveOccurred())
						Expect(user).To(Not(BeNil()))
						Expect(decision).To(Equal(httpsec.Allow))
					})
				})
			})

			Context("broker platform credentials", func() {
				const (
					currentUser      = "currentUser"
					currentPassword  = "currentPassword"
					previousUser     = "previousUsername"
					previousPassword = "previousPassword"
				)

				var req *web.Request

				BeforeEach(func() {
					authenticator.BasicAuthenticatorFunc = authenticators.BasicOSBAuthenticator
					req = &web.Request{Request: request}
					req.PathParams = map[string]string{
						osb.BrokerIDPathParam: "123",
					}
				})

				Context("When no broker platform credentials are found", func() {

					Context("platform credentials are not found", func() {
						BeforeEach(func() {
							fakeRepository.ListReturns(&types.BrokerPlatformCredentials{}, nil)
						})

						It("Should deny with error", func() {
							user, decision, err := authenticator.Authenticate(req)
							Expect(err).To(HaveOccurred())
							Expect(user).To(BeNil())
							Expect(decision).To(Equal(httpsec.Deny))
						})
					})

					Context("platform credentials are found", func() {
						BeforeEach(func() {
							fakeRepository.ListReturnsOnCall(0, &types.BrokerPlatformCredentials{}, nil)
							fakeRepository.ListReturnsOnCall(1, &types.BrokerPlatformCredentials{}, nil)
							fakeRepository.ListReturnsOnCall(2, &types.Platforms{
								Platforms: []*types.Platform{
									{
										Base: types.Base{
											ID: "id1",
										},
										Credentials: &types.Credentials{
											Basic: &types.Basic{
												Username: "username",
												Password: "password",
											},
										},
									},
								},
							}, nil)
						})

						It("Should allow", func() {
							user, decision, err := authenticator.Authenticate(req)
							Expect(err).NotTo(HaveOccurred())
							Expect(user).ToNot(BeNil())
							Expect(decision).To(Equal(httpsec.Allow))
						})
					})
				})

				Context("When getting broker platform credentials from storage fails", func() {
					expectedError := fmt.Errorf("error")

					BeforeEach(func() {
						fakeRepository.ListReturns(nil, expectedError)
					})

					It("should abstain with error", func() {
						user, decision, err := authenticator.Authenticate(req)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
						Expect(user).To(BeNil())
						Expect(decision).To(Equal(httpsec.Abstain))
					})
				})

				Context("when credentials are found in DB", func() {
					var credentialsFromDB *types.BrokerPlatformCredentials

					BeforeEach(func() {
						passwordHash, err := bcrypt.GenerateFromPassword([]byte(currentPassword), bcrypt.DefaultCost)
						Expect(err).ToNot(HaveOccurred())

						oldPasswordHash, err := bcrypt.GenerateFromPassword([]byte(previousPassword), bcrypt.DefaultCost)
						Expect(err).ToNot(HaveOccurred())

						credentialsFromDB = &types.BrokerPlatformCredentials{
							BrokerPlatformCredentials: []*types.BrokerPlatformCredential{
								{
									BrokerID:        "123",
									Username:        currentUser,
									PasswordHash:    string(passwordHash),
									OldUsername:     previousUser,
									OldPasswordHash: string(oldPasswordHash),
								},
							},
						}

						fakeRepository.ListReturnsOnCall(0, credentialsFromDB, nil)
					})

					Context("When broker platform credentials do not match", func() {
						BeforeEach(func() {
							req.SetBasicAuth("admin", "admin")
						})

						It("should deny", func() {
							user, decision, err := authenticator.Authenticate(req)
							Expect(err).To(HaveOccurred())
							Expect(user).To(BeNil())
							Expect(decision).To(Equal(httpsec.Deny))
						})
					})

					Context("When broker platform credentials match", func() {

						Context("When current credentials match", func() {
							BeforeEach(func() {
								req.SetBasicAuth(currentUser, currentPassword)
							})

							It("it should allow", func() {
								user, decision, err := authenticator.Authenticate(req)
								Expect(err).ToNot(HaveOccurred())
								Expect(user).ToNot(BeNil())
								Expect(decision).To(Equal(httpsec.Allow))
							})
						})

						Context("When old credentials match", func() {
							BeforeEach(func() {
								req.SetBasicAuth(previousUser, previousPassword)
							})

							It("it should allow", func() {
								fakeRepository.ListReturnsOnCall(0, &types.BrokerPlatformCredentials{}, nil)
								fakeRepository.ListReturnsOnCall(1, credentialsFromDB, nil)

								user, decision, err := authenticator.Authenticate(req)
								Expect(err).ToNot(HaveOccurred())
								Expect(user).ToNot(BeNil())
								Expect(decision).To(Equal(httpsec.Allow))
							})

						})

						Context("When getting platform corresponding to broker platform credentials fails", func() {
							expectedError := fmt.Errorf("error")

							BeforeEach(func() {
								req.SetBasicAuth(currentUser, currentPassword)
								fakeRepository.GetReturns(nil, expectedError)
							})

							It("should abstain with error", func() {
								user, decision, err := authenticator.Authenticate(req)
								Expect(err).To(HaveOccurred())
								Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
								Expect(user).To(BeNil())
								Expect(decision).To(Equal(httpsec.Abstain))

							})
						})

					})
				})

			})

		})
	})
})
