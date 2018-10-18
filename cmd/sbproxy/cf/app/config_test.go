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

package app

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"

	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/cloudfoundry-community/go-cfclient"
)

var _ = Describe("Config", func() {
	var (
		err    error
		config *ClientConfiguration
	)

	BeforeEach(func() {
		config = DefaultClientConfiguration()
		config.Reg.User = "user"
		config.Reg.Password = "pass"
	})

	Describe("Validate", func() {
		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).Should(HaveOccurred())
		}

		assertNoErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).ShouldNot(HaveOccurred())
		}

		Context("when config is valid", func() {
			It("returns no error", func() {
				assertNoErrorDuringValidate()
			})
		})

		Context("when address is missing", func() {
			It("returns an error", func() {
				config.Config = nil
				assertErrorDuringValidate()
			})
		})

		Context("when request timeout is missing", func() {
			It("returns an error", func() {
				config.ApiAddress = ""
				assertErrorDuringValidate()
			})
		})

		Context("when shutdown timeout is missing", func() {
			It("returns an error", func() {
				config.Reg = nil
				assertErrorDuringValidate()
			})
		})

		Context("when log level is missing", func() {
			It("returns an error", func() {
				config.Reg.User = ""
				assertErrorDuringValidate()
			})
		})

		Context("when log format  is missing", func() {
			It("returns an error", func() {
				config.Reg.Password = ""
				assertErrorDuringValidate()
			})
		})

	})

	Describe("New Configuration", func() {
		var (
			fakeEnv       *envfakes.FakeEnvironment
			creationError = fmt.Errorf("creation error")
		)

		assertErrorDuringNewConfiguration := func() {
			_, err := NewConfig(fakeEnv)
			Expect(err).Should(HaveOccurred())
		}

		BeforeEach(func() {
			fakeEnv = &envfakes.FakeEnvironment{}
		})

		Context("when unmarshaling from environment fails", func() {
			It("returns an error", func() {
				fakeEnv.UnmarshalReturns(creationError)

				assertErrorDuringNewConfiguration()
			})
		})

		Context("when unmarshaling from environment is successful", func() {
			var (
				settings Settings

				envSettings = Settings{
					Cf: &ClientConfiguration{
						Config: &cfclient.Config{
							ApiAddress:   "https://example.com",
							Username:     "user",
							Password:     "password",
							ClientID:     "clientid",
							ClientSecret: "clientsecret",
						},
						Reg: &RegistrationDetails{
							User:     "user",
							Password: "passsword",
						},
						CfClientCreateFunc: cfclient.NewClient,
					},
				}

				emptySettings = Settings{
					Cf: &ClientConfiguration{
						Reg: &RegistrationDetails{
							User:     "user",
							Password: "password",
						},
					},
				}
			)

			BeforeEach(func() {
				fakeEnv.UnmarshalReturns(nil)
				fakeEnv.UnmarshalStub = func(value interface{}) error {
					val, ok := value.(*Settings)
					if ok {
						*val = settings
					}
					return nil
				}
			})

			Context("when loaded from environment", func() {
				JustBeforeEach(func() {
					settings = envSettings
				})

				Specify("the environment values are used", func() {
					c, err := NewConfig(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(err).To(Not(HaveOccurred()))

					Expect(c.ApiAddress).Should(Equal(envSettings.Cf.ApiAddress))
					Expect(c.ClientID).Should(Equal(envSettings.Cf.ClientID))
					Expect(c.ClientSecret).Should(Equal(envSettings.Cf.ClientSecret))
					Expect(c.Username).Should(Equal(envSettings.Cf.Username))
					Expect(c.Password).Should(Equal(envSettings.Cf.Password))
				})
			})

			Context("when missing from environment", func() {
				JustBeforeEach(func() {
					settings = emptySettings
				})

				It("returns an empty config", func() {
					c, err := NewConfig(fakeEnv)
					Expect(err).To(Not(HaveOccurred()))

					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(c).Should(Equal(emptySettings.Cf))

				})
			})
		})
	})
})
