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
	"github.com/Peripli/service-manager/pkg/agents"
	"testing"
	"time"

	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/health"

	"github.com/Peripli/service-manager/api"
	cfg "github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/storage"
	. "github.com/onsi/ginkgo/v2"
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
		var fatal bool
		var failuresThreshold int64
		var interval time.Duration

		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).To(HaveOccurred())
		}

		registerIndicatorSettings := func() {
			indicatorSettings := &health.IndicatorSettings{
				Fatal:             fatal,
				FailuresThreshold: failuresThreshold,
				Interval:          interval,
			}

			config.Health.Indicators["test"] = indicatorSettings
		}

		BeforeEach(func() {
			config = cfg.DefaultSettings()
			config.Storage.URI = "postgres://postgres:postgres@localhost:5555/postgres?sslmode=disable"
			config.API.TokenIssuerURL = "http://example.com"
			config.API.ClientID = "sm"
			config.Storage.EncryptionKey = "ejHjRNHbS0NaqARSRvnweVV9zcmhQEa8"

			config.Operations.Pools = []operations.PoolSettings{
				{
					Resource: "ServiceBroker",
					Size:     5,
				},
			}

			fatal = true
			failuresThreshold = 1
			interval = 30 * time.Second
			config.Multitenancy.LabelKey = "tenant"
		})

		Context("health indicator with negative threshold", func() {
			It("should be considered invalid", func() {
				failuresThreshold = -1
				registerIndicatorSettings()
				assertErrorDuringValidate()
			})
		})

		Context("health indicator with 0 threshold", func() {
			It("should be considered invalid if it is fatal", func() {
				failuresThreshold = 0
				registerIndicatorSettings()
				assertErrorDuringValidate()
			})
		})

		Context("health indicator with 0 threshold", func() {
			It("should be considered valid if it is not fatal", func() {
				fatal = false
				failuresThreshold = 0
				registerIndicatorSettings()
				err := config.Validate()
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("health indicator with positive threshold", func() {
			It("should be considered invalid if it is not fatal", func() {
				fatal = false
				failuresThreshold = 3
				registerIndicatorSettings()
				assertErrorDuringValidate()
			})
		})

		Context("health indicator with interval less than 10", func() {
			It("should be considered invalid", func() {
				interval = 5 * time.Second
				registerIndicatorSettings()
				assertErrorDuringValidate()
			})
		})

		Context("health indicator with positive threshold and interval >= 10", func() {
			It("should be considered valid", func() {
				interval = 30 * time.Second
				failuresThreshold = 3
				registerIndicatorSettings()

				err := config.Validate()

				Expect(err).ShouldNot(HaveOccurred())
			})
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

		Context("when long request timeout is missing", func() {
			It("does not return an error", func() {
				config.Server.LongRequestTimeout = 0
				err = config.Validate()
				Expect(err).ToNot(HaveOccurred())
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

		Context("when TransactionalRepository URI is missing", func() {
			It("returns an error", func() {
				config.Storage.URI = ""
				assertErrorDuringValidate()
			})
		})

		Context("when TransactionalRepository Encryption key is missing", func() {
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

		Context("when notification queues size is 0", func() {
			It("returns an error", func() {
				config.Storage.Notification.QueuesSize = 0
				assertErrorDuringValidate()
			})
		})

		Context("when notification min reconnect interval is greater than max reconnect interval", func() {
			It("returns an error", func() {
				config.Storage.Notification.MinReconnectInterval = 100 * time.Millisecond
				config.Storage.Notification.MaxReconnectInterval = 50 * time.Millisecond
				assertErrorDuringValidate()
			})
		})

		Context("when notification keep for is < 0", func() {
			It("returns an error", func() {
				config.Storage.Notification.KeepFor = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when notification Clean interval is < 0", func() {
			It("returns an error", func() {
				config.Storage.Notification.CleanInterval = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when notification min reconnect interval is < 0", func() {
			It("returns an error", func() {
				config.Storage.Notification.MinReconnectInterval = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when operation action timeout is < 0", func() {
			It("returns an error", func() {
				config.Operations.ActionTimeout = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when operation cleanup interval is < 0", func() {
			It("returns an error", func() {
				config.Operations.CleanupInterval = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when operation scheduled deletion timeoutt is < 0", func() {
			It("returns an error", func() {
				config.Operations.ReconciliationOperationTimeout = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when operation rescheduling interval < 0", func() {
			It("returns an error", func() {
				config.Operations.ReschedulingInterval = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when operation polling interval < 0", func() {
			It("returns an error", func() {
				config.Operations.PollingInterval = -time.Second
				assertErrorDuringValidate()
			})
		})

		Context("when operation default pool size is <= 0", func() {
			It("returns an error", func() {
				config.Operations.DefaultPoolSize = 0
				assertErrorDuringValidate()
			})
		})

		Context("when operation pool size is 0", func() {
			It("returns an error", func() {
				config.Operations.Pools = []operations.PoolSettings{
					{
						Resource: "ServiceBroker",
						Size:     0,
					},
				}
				assertErrorDuringValidate()
			})
		})
		Context("when agents versions json is malformed", func() {
			It("should return an error", func() {
				config.Agents.Versions = `rsions":["1.0.0", "1.0.1", "1.0.2"],"k8s-versions":["2.0.0", "2.0.1"]}`
				assertErrorDuringValidate()
			})

		})

		Context("when multitenancy label key is empty", func() {
			It("returns an error", func() {
				config.Multitenancy.LabelKey = ""
				assertErrorDuringValidate()
			})
		})

		Context("rate limiter activated", func() {
			BeforeEach(func() {
				config.API.RateLimitingEnabled = true
			})
			When("invalid configuration specified", func() {
				It("returns error", func() {
					config.API.RateLimit = "5"
					assertErrorDuringValidate()
				})
			})
			When("empty path in configuration specified", func() {
				It("returns error", func() {
					config.API.RateLimit = "5-M:"
					assertErrorDuringValidate()
				})
			})
			When("empty element in configuration specified", func() {
				It("Returns error", func() {
					config.API.RateLimit = "5-M:/aaa,"
					assertErrorDuringValidate()
				})
			})
			When("path with multiple slashes in configuration specified", func() {
				It("returns error", func() {
					config.API.RateLimit = "5-M:///,"
					assertErrorDuringValidate()
				})
			})
			When("path not starts from slash in configuration specified", func() {
				It("returns error", func() {
					config.API.RateLimit = "5-M:v1/aaa,"
					assertErrorDuringValidate()
				})
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
						TokenIssuerURL: "http://example.com",
						ClientID:       "sm",
					},
					Agents: &agents.Settings{
						Versions: `{"cf-versions":["1.0.0", "1.0.1", "1.0.2"],"k8s-versions":["2.0.0", "2.0.1"]}`,
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
