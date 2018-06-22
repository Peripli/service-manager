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

package cf

import (
	"os"
	"testing"

	"io/ioutil"

	"github.com/Peripli/service-manager/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

const VCAP_SERVICES_VALUE = `{ "postgresql": [
   {
    "binding_name": null,
    "credentials": {
     "dbname": "smdb",
     "hostname": "10.11.2.197",
     "password": "fdb669853c9506578c357487fc7d0c0f",
     "port": "5432",
     "read_url": "jdbc:postgresql://10.11.2.192,10.11.2.193/3e546b2a3482d5de4c34                                                                                                                                   ab92f78260b9?targetServerType=preferSlave\u0026loadBalanceHosts=true",
     "uri": "postgres://9ec6640112be6ad0380ed35544db7932:fdb669853c9506578c35748                                                                                                                                   7fc7d0c0f@10.11.2.197:5432/3e546b2a3482d5de4c34ab92f78260b9",
     "username": "9ec6640112be6ad0380ed35544db7932",
     "write_url": "jdbc:postgresql://10.11.2.192,10.11.2.193/3e546b2a3482d5de4c3                                                                                                                                   4ab92f78260b9?targetServerType=master"
    },
    "instance_name": "smdb",
    "label": "postgresql",
    "name": "smdb",
    "plan": "v9.6-xsmall",
    "provider": null,
    "syslog_drain_url": null,
    "tags": [
     "postgresql",
     "relational"
    ],
    "volume_mounts": []
   }
  ]
 }`

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CF Env Suite")
}

var _ = Describe("CF Env", func() {
	var (
		env config.Environment
		err error
	)

	BeforeSuite(func() {
		Expect(ioutil.WriteFile("application.yml", []byte{}, 0640)).ShouldNot(HaveOccurred())
	})

	AfterSuite(func() {
		Expect(os.Remove("application.yml")).ShouldNot(HaveOccurred())
	})

	BeforeEach(func() {
		Expect(os.Setenv("VCAP_APPLICATION", "{}")).ShouldNot(HaveOccurred())
		Expect(os.Setenv("VCAP_SERVICES", VCAP_SERVICES_VALUE)).ShouldNot(HaveOccurred())
		Expect(os.Setenv("STORAGE_NAME", "smdb")).ShouldNot(HaveOccurred())

		env, err = config.NewEnv(pflag.CommandLine)
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("STORAGE_NAME")).ShouldNot(HaveOccurred())
	})

	Describe("Set CF env values", func() {
		Context("when VCAP_APPLICATION is missing", func() {
			It("returns no error", func() {
				Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())

				Expect(SetEnvValues(env)).ShouldNot(HaveOccurred())
				Expect(env.Get("store.uri")).Should(BeNil())
			})
		})

		Context("when VCAP_APPLICATION is present", func() {
			Context("when storage.name is missing from env", func() {
				It("returns no error", func() {
					Expect(os.Unsetenv("STORAGE_NAME")).ShouldNot(HaveOccurred())

					Expect(SetEnvValues(env)).ShouldNot(HaveOccurred())
					Expect(env.Get("storage.name")).Should(BeNil())
					Expect(env.Get("storage.uri")).Should(BeNil())

				})
			})

			Context("when storage with name storage.name is missing from VCAP_SERVICES", func() {
				It("returns error", func() {
					Expect(os.Setenv("STORAGE_NAME", "missing")).ShouldNot(HaveOccurred())

					err := SetEnvValues(env)
					Expect(SetEnvValues(env)).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not find service with name"))
				})
			})

			Context("when VCAP_SERVICES is invalid", func() {
				It("returns error", func() {
					Expect(os.Setenv("VCAP_SERVICES", "Invalid")).ShouldNot(HaveOccurred())

					Expect(SetEnvValues(env)).Should(HaveOccurred())
				})
			})

			Context("when VCAP_SERVICES is missing", func() {
				It("returns error", func() {
					Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())

					Expect(SetEnvValues(env)).Should(HaveOccurred())
				})
			})

			It("sets the storage.uri if successful", func() {
				Expect(SetEnvValues(env)).ShouldNot(HaveOccurred())

				Expect(env.Get("storage.uri")).ShouldNot(BeEmpty())
			})
		})
	})
})
