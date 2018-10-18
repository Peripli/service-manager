package app

import (
	"os"

	"github.com/Peripli/service-manager/pkg/env"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("CF Env", func() {
	const VCAP_APPLICATION = `{"instance_id":"fe98dc76ba549876543210abcd1234",
   "instance_index":0,
   "host":"0.0.0.0",
   "port":8080,
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
      "example.com"
   ],
   "version":"ab12cd34-5678-abcd-0123-abcdef987654",
   "name":"my-app",
   "uris":[  
      "example.com"
   ],
   "users":null,
   "cf_api":"https://example.com"
}`

	var (
		environment env.Environment
		err         error
	)

	BeforeEach(func() {
		Expect(os.Setenv("VCAP_APPLICATION", VCAP_APPLICATION)).ShouldNot(HaveOccurred())
		Expect(os.Setenv("VCAP_SERVICES", "{}")).ShouldNot(HaveOccurred())

		environment, err = env.New(pflag.CommandLine)
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())
	})

	Describe("Set CF environment values", func() {
		Context("when VCAP_APPLICATION is missing", func() {
			It("returns no error", func() {
				Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())

				Expect(SetCFOverrides(environment)).ShouldNot(HaveOccurred())
				Expect(environment.Get("server.host")).Should(BeNil())
				Expect(environment.Get("server.port")).Should(BeNil())
				Expect(environment.Get("cf.api")).Should(BeNil())

			})
		})

		Context("when VCAP_APPLICATION is present", func() {
			It("sets self_url", func() {
				Expect(SetCFOverrides(environment)).ShouldNot(HaveOccurred())

				Expect(environment.Get("self_url")).To(Equal("https://example.com"))
			})

			It("sets server.port", func() {
				Expect(SetCFOverrides(environment)).ShouldNot(HaveOccurred())

				Expect(environment.Get("server.port")).To(Equal(8080))
			})

			It("sets cf.client.apiAddress", func() {
				Expect(SetCFOverrides(environment)).ShouldNot(HaveOccurred())

				Expect(environment.Get("cf.client.apiAddress")).To(Equal("https://example.com"))
			})
		})
	})
})
