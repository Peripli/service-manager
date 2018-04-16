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

package server_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	. "github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/server/serverfakes"
	"strconv"
	"time"
)

var _ = Describe("config", func() {

	var (
		err    error
		config *Config
	)

	Describe("Validate", func() {

		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			config = DefaultConfiguration()
			config.DbURI = "postgres://postgres:postgres@localhost:5555/postgres?sslmode=disable"
		})

		Context("when config is valid", func() {
			It("returns no error", func() {
				err = config.Validate()
				Expect(err).To(Not(HaveOccurred()))
			})
		})

		Context("when address is missing", func() {
			It("returns an error", func() {
				config.Address = ""
				assertErrorDuringValidate()
			})
		})

		Context("when request timeout is missing", func() {
			It("returns an error", func() {
				config.RequestTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when shutdown timeout is missing", func() {
			It("returns an error", func() {
				config.ShutdownTimeout = 0
				assertErrorDuringValidate()
			})
		})

		Context("when log level is missing", func() {
			It("returns an error", func() {
				config.LogLevel = ""
				assertErrorDuringValidate()
			})
		})

		Context("when log format  is missing", func() {
			It("returns an error", func() {
				config.LogFormat = ""
				assertErrorDuringValidate()
			})
		})

		Context("when DB URI is missing", func() {
			It("returns an error", func() {
				config.DbURI = ""
				assertErrorDuringValidate()
			})
		})

	})

	Describe("New Configuration", func() {

		var (
			fakeEnv       *serverfakes.FakeEnvironment
			creationError = fmt.Errorf("creation error")
		)

		assertErrorDuringNewConfiguration := func() {
			_, err := NewConfiguration(fakeEnv)
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
			fakeEnv = &serverfakes.FakeEnvironment{}
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

		Context("when loading and unmarshaling from environment are successful", func() {

			var (
				settings Settings

				envSettings = Settings{
					Server: &AppSettings{
						Port:            8080,
						ShutdownTimeout: 5000,
						RequestTimeout:  5000,
					},
					Db: &DbSettings{
						URI: "dbUri",
					},
					Log: &LogSettings{
						Format: "text",
						Level:  "debug",
					},
				}

				emptySettings = Settings{
					Server: &AppSettings{},
					Db:     &DbSettings{},
					Log:    &LogSettings{},
				}
			)

			assertEnvironmentLoadedAndUnmarshaled := func() {
				Expect(fakeEnv.LoadCallCount()).To(Equal(1))
				Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))
			}

			BeforeEach(func() {
				fakeEnv.LoadReturns(nil)
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
					c, err := NewConfiguration(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					assertEnvironmentLoadedAndUnmarshaled()

					Expect(err).To(Not(HaveOccurred()))

					Expect(c.Address).Should(Equal(":" + strconv.Itoa(envSettings.Server.Port)))
					Expect(c.RequestTimeout).Should(Equal(time.Millisecond * time.Duration(envSettings.Server.RequestTimeout)))
					Expect(c.ShutdownTimeout).Should(Equal(time.Millisecond * time.Duration(envSettings.Server.ShutdownTimeout)))
					Expect(c.LogLevel).Should(Equal(envSettings.Log.Level))
					Expect(c.LogFormat).Should(Equal(envSettings.Log.Format))
					Expect(c.DbURI).Should(Equal(envSettings.Db.URI))
				})
			})

			Context("when missing from environment", func() {
				JustBeforeEach(func() {
					settings = emptySettings
				})

				Specify("the default value is used", func() {
					c, err := NewConfiguration(fakeEnv)
					Expect(err).To(Not(HaveOccurred()))

					assertEnvironmentLoadedAndUnmarshaled()

					Expect(c).Should(Equal(DefaultConfiguration()))
				})
			})
		})
	})
})
