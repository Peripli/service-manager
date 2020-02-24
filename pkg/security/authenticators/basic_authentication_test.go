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
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"

	"github.com/Peripli/service-manager/storage/storagefakes"

	"github.com/Peripli/service-manager/pkg/security/authenticators"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

		request, err = http.NewRequest(http.MethodGet, "https://example.com", nil)
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
	})
})
