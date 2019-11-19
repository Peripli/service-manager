package cf_test

import (
	"github.com/Peripli/service-manager/pkg/agent"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"

	"github.com/Peripli/service-manager/cmd/cf-agent/cf"
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/cloudfoundry-community/go-cfclient"
)

var _ = Describe("Config", func() {
	var (
		err      error
		settings *cf.Settings
	)

	BeforeEach(func() {
		settings = cf.DefaultSettings()

		settings.Settings.Sm.URL = "url"
		settings.Settings.Sm.User = "user"
		settings.Settings.Sm.Password = "password"
		settings.Settings.Reconcile.URL = "url"
		settings.Settings.Reconcile.LegacyURL = "legacyurl"
		settings.CF.ApiAddress = "http://apiaddress.com"
	})

	Describe("Validate", func() {
		assertErrorDuringValidate := func() {
			err = settings.Validate()
			Expect(err).Should(HaveOccurred())
		}

		assertNoErrorDuringValidate := func() {
			err = settings.Validate()
			Expect(err).ShouldNot(HaveOccurred())
		}

		Context("when config is valid", func() {
			It("returns no error", func() {
				assertNoErrorDuringValidate()
			})
		})

		Context("when address is missing", func() {
			It("returns an error", func() {
				settings.CF.Config = nil
				assertErrorDuringValidate()
			})
		})

		Context("when request timeout is missing", func() {
			It("returns an error", func() {
				settings.CF.ApiAddress = ""
				assertErrorDuringValidate()
			})
		})

		Context("when shutdown timeout is missing", func() {
			It("returns an error", func() {
				settings.CF = nil
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
			_, err := cf.NewConfig(fakeEnv)
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
				settings cf.Settings

				envSettings = cf.Settings{
					CF: &cf.ClientConfiguration{
						Config: &cfclient.Config{
							ApiAddress:   "https://example.com",
							Username:     "user",
							Password:     "password",
							ClientID:     "clientid",
							ClientSecret: "clientsecret",
						},
						CFClientProvider: cfclient.NewClient,
					},
					Settings: *agent.DefaultSettings(),
				}

				emptySettings = cf.Settings{
					CF:       &cf.ClientConfiguration{},
					Settings: *agent.DefaultSettings(),
				}
			)

			BeforeEach(func() {
				fakeEnv.UnmarshalReturns(nil)
				fakeEnv.UnmarshalStub = func(value interface{}) error {
					val, ok := value.(*cf.Settings)
					if ok {
						*val = settings
					}
					return nil
				}

				envSettings.Reconcile.URL = "http://10.0.2.2"
				emptySettings.Reconcile.URL = "http://10.0.2.2"
			})

			Context("when loaded from environment", func() {
				JustBeforeEach(func() {
					settings = envSettings
				})

				Specify("the environment values are used", func() {
					c, err := cf.NewConfig(fakeEnv)

					Expect(err).To(Not(HaveOccurred()))
					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(err).To(Not(HaveOccurred()))

					Expect(c.CF.ApiAddress).Should(Equal(envSettings.CF.ApiAddress))
					Expect(c.CF.ClientID).Should(Equal(envSettings.CF.ClientID))
					Expect(c.CF.ClientSecret).Should(Equal(envSettings.CF.ClientSecret))
					Expect(c.CF.Username).Should(Equal(envSettings.CF.Username))
					Expect(c.CF.Password).Should(Equal(envSettings.CF.Password))
				})
			})

			Context("when missing from environment", func() {
				JustBeforeEach(func() {
					settings = emptySettings
				})

				It("returns an empty config", func() {
					c, err := cf.NewConfig(fakeEnv)
					Expect(err).To(Not(HaveOccurred()))

					Expect(fakeEnv.UnmarshalCallCount()).To(Equal(1))

					Expect(c.CF).Should(Equal(emptySettings.CF))

				})
			})
		})
	})
})
