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
	var delegate config.Environment
	var env config.Environment

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

		delegate = config.NewEnv()
		env = NewEnv(delegate)
	})

	AfterEach(func() {
		Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("STORAGE_NAME")).ShouldNot(HaveOccurred())
	})

	Describe("New", func() {
		verifyEnvIsCFEnv := func(isCFEnv bool) {
			env = NewEnv(delegate)
			_, ok := env.(*cfEnvironment)

			Expect(ok).To(Equal(isCFEnv))
		}

		Context("when VCAP_APPLICATION is missing", func() {
			It("returns non cf env", func() {
				Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())
				verifyEnvIsCFEnv(false)
			})
		})

		Context("when VCAP_APPLICATION is present", func() {
			It("returns cf env", func() {
				verifyEnvIsCFEnv(true)
			})
		})
	})

	Describe("Load", func() {
		Context("with missing STORAGE_name", func() {
			It("succeeds", func() {
				Expect(os.Unsetenv("STORAGE_NAME")).ShouldNot(HaveOccurred())
				err := env.Load()

				Expect(err).ShouldNot(HaveOccurred())
				Expect(env.Get("storage_name")).Should(BeNil())
				Expect(env.Get("storage_uri")).Should(BeNil())

			})
		})

		Context("with missing postgresql service", func() {
			It("returns error", func() {
				Expect(os.Setenv("STORAGE_NAME", "missing")).ShouldNot(HaveOccurred())
				err := env.Load()

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("could not find service with name"))
			})
		})

		Context("with invalid VCAP_SERVICES", func() {
			It("returns error", func() {
				Expect(os.Setenv("VCAP_SERVICES", "Invalid")).ShouldNot(HaveOccurred())

				Expect(env.Load()).Should(HaveOccurred())
			})
		})

		Context("with missing VCAP_SERVICES", func() {
			It("returns error", func() {
				Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())

				Expect(env.Load()).Should(HaveOccurred())
			})
		})

		It("sets the STORAGE_uri error", func() {
			Expect(env.Load()).ShouldNot(HaveOccurred())
			Expect(env.Get("storage_uri")).ShouldNot(BeEmpty())
		})
	})
})
