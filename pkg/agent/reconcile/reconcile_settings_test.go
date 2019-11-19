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

package reconcile_test

import (
	"github.com/Peripli/service-manager/pkg/agent/reconcile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func validSettings() *reconcile.Settings {
	settings := reconcile.DefaultSettings()
	settings.URL = "http://localhost:8080"
	settings.LegacyURL = "http://localhost:8080"
	return settings
}

var _ = Describe("Reconcile", func() {
	Describe("Settings", func() {
		Describe("Validate", func() {
			Context("when all properties are set correctly", func() {
				It("no error is returned", func() {
					Expect(validSettings().Validate()).ShouldNot(HaveOccurred())
				})
			})

			Context("when LegacyURL is missing", func() {
				It("returns an error", func() {
					settings := validSettings()
					settings.LegacyURL = ""
					Expect(settings.Validate()).Should(HaveOccurred())
				})
			})

			Context("when URL is missing", func() {
				It("returns an error", func() {
					settings := validSettings()
					settings.URL = ""
					Expect(settings.Validate()).Should(HaveOccurred())
				})
			})

			Context("when max parallel requests is zero", func() {
				It("returns an error", func() {
					settings := validSettings()
					settings.MaxParallelRequests = 0
					Expect(settings.Validate()).Should(HaveOccurred())
				})
			})
		})
	})
})
