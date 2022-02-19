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

package env

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const VCAP_SERVICES_VALUE = `{ "postgresql": [
   {
    "binding_name": null,
    "credentials": {
     "dbname": "smdb",
     "hostname": "10.11.5.197",
     "password": "fdb669853c9506578c357487fc7d0c0f",
     "port": "5432",
     "read_url": "jdbc:postgresql://10.11.5.192,10.11.5.193/3e546b2a3482d5de4c34                                                                                                                                   ab92f78260b9?targetServerType=preferSlave\u0026loadBalanceHosts=true",
     "uri": "postgres://9ec6640112be6ad0380ed35544db7932:fdb669853c9506578c35748                                                                                                                                   7fc7d0c0f@10.11.oidc_authn.197:5432/3e546b2a3482d5de4c34ab92f78260b9",
     "username": "9ec6640112be6ad0380ed35544db7932",
     "write_url": "jdbc:postgresql://10.11.5.192,10.11.5.193/3e546b2a3482d5de4c3                                                                                                                                   4ab92f78260b9?targetServerType=master"
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

var _ = Describe("CF Env", func() {
	var (
		environment Environment
		err         error
	)

	BeforeEach(func() {
		Expect(os.Setenv("VCAP_APPLICATION", "{}")).ShouldNot(HaveOccurred())
		Expect(os.Setenv("VCAP_SERVICES", VCAP_SERVICES_VALUE)).ShouldNot(HaveOccurred())
		Expect(os.Setenv("STORAGE_NAME", "smdb")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("STORAGE_URI")).ShouldNot(HaveOccurred())

		environment, err = New(context.TODO(), EmptyFlagSet())
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv("STORAGE_NAME")).ShouldNot(HaveOccurred())
	})

	Describe("Set CF environment values", func() {
		Context("when VCAP_APPLICATION is missing", func() {
			It("returns no error", func() {
				Expect(os.Unsetenv("VCAP_APPLICATION")).ShouldNot(HaveOccurred())

				Expect(setCFOverrides(environment)).ShouldNot(HaveOccurred())
				Expect(environment.Get("store.uri")).Should(BeNil())
			})
		})

		Context("when VCAP_APPLICATION is present", func() {
			Context("when storage.name is missing from environment", func() {
				It("returns no error", func() {
					Expect(os.Unsetenv("STORAGE_NAME")).ShouldNot(HaveOccurred())

					Expect(setCFOverrides(environment)).ShouldNot(HaveOccurred())
					Expect(environment.Get("storage.name")).Should(BeNil())
					Expect(environment.Get("storage.uri")).Should(BeNil())

				})
			})

			Context("when storage with name storage.name is missing from VCAP_SERVICES", func() {
				It("returns error", func() {
					Expect(os.Setenv("STORAGE_NAME", "missing")).ShouldNot(HaveOccurred())

					err := setCFOverrides(environment)
					Expect(setCFOverrides(environment)).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not find service with name"))
				})
			})

			Context("when VCAP_SERVICES is invalid", func() {
				It("returns error", func() {
					Expect(os.Setenv("VCAP_SERVICES", "Invalid")).ShouldNot(HaveOccurred())

					Expect(setCFOverrides(environment)).Should(HaveOccurred())
				})
			})

			Context("when VCAP_SERVICES is missing", func() {
				It("returns error", func() {
					Expect(os.Unsetenv("VCAP_SERVICES")).ShouldNot(HaveOccurred())

					Expect(setCFOverrides(environment)).Should(HaveOccurred())
				})
			})

			It("sets the storage.uri if successful", func() {
				Expect(setCFOverrides(environment)).ShouldNot(HaveOccurred())

				Expect(environment.Get("storage.uri")).ShouldNot(BeEmpty())
			})
		})
	})
})
