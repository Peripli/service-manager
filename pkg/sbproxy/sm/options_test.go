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

package sm

import (
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/fatih/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var (
		err    error
		config *Settings
	)

	Describe("New", func() {
		var (
			fakeEnv       *envfakes.FakeEnvironment
			creationError = fmt.Errorf("creation error")
		)

		assertErrorDuringNewConfiguration := func() {
			_, err := NewSettings(fakeEnv)
			Expect(err).To(HaveOccurred())
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

		Context("when unmarshalling is successful", func() {
			var (
				validSettings *Settings
				emptySettings *Settings
			)

			BeforeEach(func() {
				emptySettings = &Settings{}
				validSettings = &Settings{
					User:              "admin",
					Password:          "admin",
					OSBAPIPath:        "/osb",
					RequestTimeout:    5 * time.Second,
					ResyncPeriod:      5 * time.Minute,
					SkipSSLValidation: true,
					Transport:         nil,
				}
				fakeEnv.UnmarshalReturns(nil)

				fakeEnv.UnmarshalStub = func(value interface{}) error {
					field := structs.New(value).Field("Sm")
					val, ok := field.Value().(*Settings)
					if ok {
						*val = *config
					}
					return nil
				}
			})
			Context("when loaded from environment", func() {
				JustBeforeEach(func() {
					config = validSettings
				})

				It("uses the config values from env", func() {
					c, err := NewSettings(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(err).To(Not(HaveOccurred()))

					Expect(c).Should(Equal(validSettings))
				})
			})

			Context("when missing from environment", func() {
				JustBeforeEach(func() {
					config = emptySettings
				})

				It("returns an empty config", func() {
					c, err := NewSettings(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(c).Should(Equal(emptySettings))
				})
			})
		})
	})

	Describe("Validate", func() {
		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			config = DefaultSettings()
			config.User = "admin"
			config.Password = "admin"
			config.OSBAPIPath = "/osb"
			config.URL = "https://example.com"
		})

		Context("when config is valid", func() {
			It("returns no error", func() {
				err = config.Validate()
				Expect(err).To(Not(HaveOccurred()))
			})
		})

		Context("when request timeout is missing", func() {
			It("returns an error", func() {
				config.RequestTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when URL is missing", func() {
			It("returns an error", func() {
				config.URL = ""
				assertErrorDuringValidate()
			})
		})

		Context("when OSB API is missing", func() {
			It("returns an error", func() {
				config.OSBAPIPath = ""
				assertErrorDuringValidate()
			})
		})

		Context("when resync period is missing", func() {
			It("returns an error", func() {
				config.ResyncPeriod = 0
				assertErrorDuringValidate()
			})
		})

		Context("when host is missing", func() {
			It("returns an error", func() {
				config.OSBAPIPath = ""
				assertErrorDuringValidate()
			})
		})

		Context("when user is missing", func() {
			It("returns an error", func() {
				config.User = ""
				assertErrorDuringValidate()
			})
		})

		Context("when password is missing", func() {
			It("returns an error", func() {
				config.Password = ""
				assertErrorDuringValidate()
			})
		})
	})
})
