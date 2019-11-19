package cf_test

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/cmd/cf-agent/cf"

	"github.com/Peripli/service-manager/pkg/server"

	"github.com/Peripli/service-manager/pkg/env"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Env", func() {

	const (
		overridenPort   = 8888
		overridenAppURI = "overriden-uri.com"
		overridenCFAPI  = "https://overriden-cf-api"
	)

	var (
		environment              env.Environment
		err                      error
		vcapApplication          string
		additionalPFlagProviders []func(set *pflag.FlagSet)
	)

	BeforeEach(func() {
		additionalPFlagProviders = make([]func(set *pflag.FlagSet), 0)
		vcapApplication = fmt.Sprintf(`{
   "instance_id":"fe98dc76ba549876543210abcd1234",
   "instance_index":0,
   "host":"0.0.0.0",
   "port":%d,
   "started_at":"2013-08-12 00:05:29 +0000",
   "started_at_timestamp":1376265929,
   "start":"2013-08-12 00:05:29 +0000",
   "state_timestamp":1376265929,
   "limits":{  
      "mem":512,
      "disk":1024,
      "fds":16384
   },
   "application_version":"ab12cd34-5678-abcd-0123-abcdef987654",
   "application_name":"styx-james",
   "application_uris":[  
      "%s"
   ],
   "version":"ab12cd34-5678-abcd-0123-abcdef987654",
   "name":"my-app",
   "uris":[  
      "example.com"
   ],
   "users":null,
   "cf_api":"%s"
}`, overridenPort, overridenAppURI, overridenCFAPI)
		Expect(os.Setenv("VCAP_APPLICATION", vcapApplication)).ShouldNot(HaveOccurred())
		Expect(os.Setenv("VCAP_SERVICES", "{}")).ShouldNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		environment, err = cf.DefaultEnv(context.TODO(), additionalPFlagProviders...)
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())
	})

	Describe("Set CF environment values", func() {
		Context("when VCAP_APPLICATION is missing", func() {
			BeforeEach(func() {
				Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())
			})

			It("CF overrides are not applied", func() {
				Expect(environment.Get("app.legacy_url")).Should(Equal(server.DefaultSettings().Host))
				Expect(environment.Get("server.port")).Should(Equal(server.DefaultSettings().Port))
				Expect(environment.Get("cf.client.apiAddress")).Should(Equal(cf.DefaultClientConfiguration().ApiAddress))
			})
		})

		Context("when VCAP_APPLICATION is present", func() {
			Context("no values are already set", func() {

				It("sets app.legacy_url", func() {
					Expect(environment.Get("app.legacy_url")).To(Equal("https://" + overridenAppURI))
				})

				It("does not set server.port as it has a default value", func() {
					Expect(environment.Get("server.port")).To(Equal(server.DefaultSettings().Port))
				})

				It("sets cf.client.apiAddress", func() {
					Expect(environment.Get("cf.client.apiAddress")).To(Equal(overridenCFAPI))
				})

				It("unmarshals correctly", func() {
					settings, err := cf.NewConfig(environment)
					Expect(err).ToNot(HaveOccurred())
					Expect(settings.Reconcile.LegacyURL).To(Equal("https://" + overridenAppURI))
					Expect(settings.Server.Port).To(Equal(server.DefaultSettings().Port))
					Expect(settings.CF.ApiAddress).To(Equal(overridenCFAPI))
				})
			})

			Context("when explicit values are set", func() {
				BeforeEach(func() {
					Expect(os.Setenv("APP_LEGACY_URL", "https://explicit-url.com")).ToNot(HaveOccurred())
					Expect(os.Setenv("SERVER_PORT", "8000")).ToNot(HaveOccurred())
					Expect(os.Setenv("CF_CLIENT_APIADDRESS", "https://explicit-cf-url.com")).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					Expect(os.Unsetenv("APP_LEGACY_URL")).ToNot(HaveOccurred())
					Expect(os.Unsetenv("SERVER_PORT")).ToNot(HaveOccurred())
					Expect(os.Unsetenv("CF_CLIENT_APIADDRESS")).ToNot(HaveOccurred())
				})

				It("doesn't override app.url", func() {
					Expect(environment.Get("app.legacy_url")).To(Equal("https://explicit-url.com"))
				})

				It("doesn't override server.port", func() {
					Expect(environment.Get("server.port")).To(Equal("8000"))
				})

				It("doesn't override cf.client.apiAddress", func() {
					Expect(environment.Get("cf.client.apiAddress")).To(Equal("https://explicit-cf-url.com"))
				})

				It("unmarshals correctly without overriding", func() {
					settings, err := cf.NewConfig(environment)
					Expect(err).ToNot(HaveOccurred())
					Expect(settings.Reconcile.LegacyURL).To(Equal("https://explicit-url.com"))
					Expect(settings.Server.Port).To(Equal(8000))
					Expect(settings.CF.ApiAddress).To(Equal("https://explicit-cf-url.com"))
				})
			})
		})
	})
})
