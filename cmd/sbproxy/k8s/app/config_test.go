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

package app

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubernetes Broker Proxy", func() {
	Describe("Config", func() {
		Describe("Validation", func() {
			var config *ClientConfiguration

			BeforeEach(func() {
				config = defaultClientConfiguration()
				config.Reg.User = "abc"
				config.Reg.Password = "abc"
				config.Reg.Secret.Name = "abc"
				config.Reg.Secret.Namespace = "abc"
			})

			Context("when all properties available", func() {
				It("should return nil", func() {
					err := config.Validate()
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("when ClientCreateFunc is missing", func() {
				It("should fail", func() {
					config.K8sClientCreateFunc = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S ClientCreateFunc missing"))
				})
			})

			Context("when LibraryConfig is missing", func() {
				It("should fail", func() {
					config.Client = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client configuration missing"))
				})
			})

			Context("when LibraryConfig.Timeout is missing", func() {
				It("should fail", func() {
					config.Client.Timeout = 0
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client configuration timeout missing"))
				})
			})

			Context("when Reg is missing", func() {
				It("should fail", func() {
					config.Reg = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S broker registration configuration missing"))
				})
			})

			Context("when Reg user is missing", func() {
				It("should fail", func() {
					config.Reg.User = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S broker registration credentials missing"))
				})
			})

			Context("when Reg password is missing", func() {
				It("should fail", func() {
					config.Reg.Password = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S broker registration credentials missing"))
				})
			})

			Context("when Reg secret is missing", func() {
				It("should fail", func() {
					config.Reg.Secret = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S secret configuration for broker registration missing"))
				})
			})

			Context("when Reg secret Name is missing", func() {
				It("should fail", func() {
					config.Reg.Secret.Name = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Properties of K8S secret configuration for broker registration missing"))
				})
			})

			Context("when Reg secret Namespace is missing", func() {
				It("should fail", func() {
					config.Reg.Secret.Namespace = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Properties of K8S secret configuration for broker registration missing"))
				})
			})

		})
	})
})
