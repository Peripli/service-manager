package server_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	. "github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/server/serverfakes"
	"time"
	"strconv"
)

var _ bool = Describe("config", func() {

	var (
		config = DefaultConfiguration()
		err    error
	)

	Describe("Validate", func() {

		assertErrorDuringValidate := func() {
			err = config.Validate()
			Expect(err).To(HaveOccurred())
		}

		BeforeEach(func() {
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

			assertEnvironmentLoadedAndUnmarshaled := func() {
				Expect(fakeEnv.LoadCallCount()).To(Equal(1))
				Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))
			}

			var (
				settings *Settings

				envSettings = &Settings{
					Server: &ServerSettings{
						Port: 8080,
						ShutdownTimeout: 5000,
						RequestTimeout: 5000,

					},
					Db: &DbSettings{
						URI: "dbUri",
					},
					Log: &LogSettings{
						Format: "text",
						Level: "debug",
					},
				}

				emptySettings = &Settings{
					Server: &ServerSettings{
					},
					Db: &DbSettings{
					},
					Log: &LogSettings{
					},
				}
			)

			BeforeEach(func() {
				fakeEnv.LoadReturns(nil)
				fakeEnv.UnmarshalReturns(nil)
				fakeEnv.UnmarshalStub = func(value interface{}) error {
					value = settings
					return nil
				}
			})

				Context("when loaded from environment", func() {
					JustBeforeEach(func() {
						settings = envSettings
					})

					XSpecify("the environment values are used", func() {
						c, err := NewConfiguration(fakeEnv)

						Expect(err).To(Not(HaveOccurred()))
						assertEnvironmentLoadedAndUnmarshaled()

						Expect(err).To(Not(HaveOccurred()))

						Expect(c.Address).Should(Equal(":" + strconv.Itoa(envSettings.Server.Port)))
						Expect(c.RequestTimeout).Should(Equal(time.Duration(envSettings.Server.RequestTimeout)))
						Expect(c.ShutdownTimeout).Should(Equal(time.Duration(envSettings.Server.ShutdownTimeout)))
						Expect(c.LogLevel).Should(Equal(envSettings.Log.Level))
						Expect(c.LogFormat).Should(Equal(envSettings.Log.Format))
						Expect(c.DbURI).Should(Equal(envSettings.Db.URI))
					})
				})

			Context("when missing from environment", func() {
				JustBeforeEach(func() {
					settings = emptySettings
				})

				XSpecify("the default value is used", func() {
					c, err := NewConfiguration(fakeEnv)
					Expect(err).To(Not(HaveOccurred()))

					assertEnvironmentLoadedAndUnmarshaled()

					Expect(c).Should(Equal(DefaultConfiguration()))
				})
			})
		})
	})
})

