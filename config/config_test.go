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
	"testing"

	"github.com/Peripli/service-manager/api"
	cfg "github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/storage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("config", func() {

	var (
		err    error
		config *cfg.Settings
	)

	Describe("Validate", func() {
		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			config = cfg.DefaultSettings()
			config.Storage.URI = "postgres://postgres:postgres@localhost:5555/postgres?sslmode=disable"
			config.API.TokenIssuerURL = "http://example.com"
			config.API.ClientID = "sm"
			config.API.SkipSSLValidation = true
			config.Storage.EncryptionKey = "ejHjRNHbS0NaqARSRvnweVV9zcmhQEa8"
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

		Context("when Repository URI is missing", func() {
			It("returns an error", func() {
				config.Storage.URI = ""
				assertErrorDuringValidate()
			})
		})

		Context("when Repository Encryption key is missing", func() {
			It("returns an error", func() {
				config.Storage.EncryptionKey = ""
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

	Describe("New", func() {
		var (
			fakeEnv       *envfakes.FakeEnvironment
			creationError = fmt.Errorf("creation error")
		)

		assertErrorDuringNewConfiguration := func() {
			_, err := cfg.New(fakeEnv)
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

		Context("when creating is successful", func() {

			var (
				configuration cfg.Settings

				envConfig = cfg.Settings{
					Server: &server.Settings{
						Port:            8080,
						ShutdownTimeout: 5000,
						RequestTimeout:  5000,
					},
					Storage: &storage.Settings{
						URI: "dbUri",
					},
					Log: &log.Settings{
						Format: "text",
						Level:  "debug",
					},
					API: &api.Settings{
						TokenIssuerURL:    "http://example.com",
						ClientID:          "sm",
						SkipSSLValidation: false,
					},
				}

				emptyConfig = cfg.Settings{}
			)

			BeforeEach(func() {
				fakeEnv.UnmarshalReturns(nil)

				fakeEnv.UnmarshalStub = func(value interface{}) error {
					val, ok := value.(*cfg.Settings)
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
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

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
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(c).Should(Equal(&emptyConfig))
				})
			})
		})
	})
})
