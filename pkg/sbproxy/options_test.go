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

package sbproxy

import (
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/sbproxy/reconcile"
	"github.com/Peripli/service-manager/pkg/sbproxy/sm"
	"github.com/Peripli/service-manager/pkg/server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	var (
		config *Settings
		err    error
	)

	Describe("NewSettings", func() {
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
					Server: &server.Settings{
						Port:            8080,
						RequestTimeout:  5 * time.Second,
						ShutdownTimeout: 5 * time.Second,
					},
					Log: &log.Settings{
						Level:  "debug",
						Format: "text",
					},
					Sm: &sm.Settings{
						User:              "admin",
						Password:          "admin",
						URL:               "https://sm.com",
						OSBAPIPath:        "/osb",
						RequestTimeout:    5 * time.Second,
						ResyncPeriod:      5 * time.Minute,
						SkipSSLValidation: true,
					},
					Reconcile: &reconcile.Settings{
						URL: "https://appurl.com",
					},
				}
				fakeEnv.UnmarshalReturns(nil)

				fakeEnv.UnmarshalStub = func(value interface{}) error {
					val, ok := value.(*Settings)
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
			config = &Settings{
				Server: &server.Settings{
					Port:            8080,
					RequestTimeout:  5 * time.Second,
					ShutdownTimeout: 5 * time.Second,
				},
				Log: &log.Settings{
					Level:  "debug",
					Format: "text",
				},
				Sm: &sm.Settings{
					User:              "admin",
					Password:          "admin",
					URL:               "https://sm.com",
					OSBAPIPath:        "/osb",
					RequestTimeout:    5 * time.Second,
					ResyncPeriod:      5 * time.Minute,
					SkipSSLValidation: true,
				},
				Reconcile: &reconcile.Settings{
					URL: "https://appurl.com",
				},
			}
		})
		Context("when app.url is missing", func() {
			It("returns an error", func() {
				config.Reconcile.URL = ""
				assertErrorDuringValidate()
			})
		})

		Context("when server config is invalid", func() {
			It("returns an error", func() {
				config.Server.RequestTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when log config is invalid", func() {
			It("returns an error", func() {
				config.Log.Format = ""
				assertErrorDuringValidate()
			})
		})

		Context("when sm config is invalid", func() {
			It("returns an error", func() {
				config.Sm.URL = ""
				assertErrorDuringValidate()
			})
		})
	})
})
