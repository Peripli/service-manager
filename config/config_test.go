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

package config_test

import (
	"fmt"

	"github.com/Peripli/service-manager/api"
	cfg "github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/config/configfakes"
	"github.com/Peripli/service-manager/log"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("config", func() {

	var (
		err    error
		config *cfg.Config
	)

	Describe("Validate", func() {

		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			config = cfg.DefaultConfig()
			config.Storage.URI = "postgres://postgres:postgres@localhost:5555/postgres?sslmode=disable"
			config.API.TokenIssuerURL = "http://example.com"
		})

		Context("when config is valid", func() {
			It("returns no error", func() {
				err = config.Validate()
				Expect(err).To(Not(HaveOccurred()))
			})
		})

		Context("when port is missing", func() {
			It("returns an error", func() {
				config.Server.Port = 0
				assertErrorDuringValidate()
			})
		})

		Context("when request timeout is missing", func() {
			It("returns an error", func() {
				config.Server.RequestTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when shutdown timeout is missing", func() {
			It("returns an error", func() {
				config.Server.ShutdownTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when log level is missing", func() {
			It("returns an error", func() {
				config.Log.Level = ""
				assertErrorDuringValidate()
			})
		})

		Context("when log format  is missing", func() {
			It("returns an error", func() {
				config.Log.Format = ""
				assertErrorDuringValidate()
			})
		})

		Context("when Storage URI is missing", func() {
			It("returns an error", func() {
				config.Storage.URI = ""
				assertErrorDuringValidate()
			})
		})

		Context("when API token issuer URL is missing", func() {
			It("returns an error", func() {
				config.API.TokenIssuerURL = ""
				assertErrorDuringValidate()
			})
		})
	})

	Describe("New Config", func() {

		var (
			fakeEnv       *configfakes.FakeEnvironment
			creationError = fmt.Errorf("creation error")
		)

		assertErrorDuringNewConfiguration := func() {
			_, err := cfg.New(fakeEnv)
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			fakeEnv = &configfakes.FakeEnvironment{}
		})

		Context("when loading from environment fails", func() {
			It("returns an error", func() {
				fakeEnv.LoadReturns(creationError)

				assertErrorDuringNewConfiguration()
			})
		})

		Context("when unmarshaling from environment fails", func() {
			It("returns an error", func() {
				fakeEnv.UnmarshalReturns(creationError)

				assertErrorDuringNewConfiguration()
			})
		})

		Context("when binding pflags fails", func() {
			It("returns an error", func() {
				fakeEnv.CreatePFlagsReturns(creationError)

				assertErrorDuringNewConfiguration()
			})
		})

		Context("when loading and unmarshaling from environment are successful", func() {

			var (
				configuration cfg.Config

				envConfig = cfg.Config{
					Server: server.Settings{
						Port:            8080,
						ShutdownTimeout: 5000,
						RequestTimeout:  5000,
					},
					Storage: storage.Settings{
						URI: "dbUri",
					},
					Log: log.Settings{
						Format: "text",
						Level:  "debug",
					},
					API: api.Settings{
						TokenIssuerURL: "http://example.com",
					},
				}

				emptyConfig = cfg.Config{}
			)

			assertEnvironmentLoadedAndUnmarshaled := func() {
				Expect(fakeEnv.LoadCallCount()).To(Equal(1))
				Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))
				Expect(fakeEnv.CreatePFlagsCallCount()).To(Equal(1))
			}

			BeforeEach(func() {
				fakeEnv.LoadReturns(nil)
				fakeEnv.UnmarshalReturns(nil)
				fakeEnv.CreatePFlagsReturns(nil)

				fakeEnv.UnmarshalStub = func(value interface{}) error {
					val, ok := value.(*cfg.Config)
					if ok {
						*val = configuration
					}
					return nil
				}
			})

			Context("when loaded from environment", func() {
				JustBeforeEach(func() {
					configuration = envConfig
				})

				It("uses the config values from env", func() {
					c, err := cfg.New(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					assertEnvironmentLoadedAndUnmarshaled()

					Expect(err).To(Not(HaveOccurred()))

					Expect(c).Should(Equal(&envConfig))
				})
			})

			Context("when missing from environment", func() {
				JustBeforeEach(func() {
					configuration = emptyConfig
				})

				It("returns an empty config", func() {
					c, err := cfg.New(fakeEnv)
					Expect(err).To(Not(HaveOccurred()))

					assertEnvironmentLoadedAndUnmarshaled()

					Expect(c).Should(Equal(&emptyConfig))
				})
			})
		})
	})
})
