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

package postgres

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Postgres Translator", func() {

	Describe("translate", func() {

		var translator = NewLabelTranslator()

		Context("Called with valid input", func() {
			It("Should return proper result", func() {
				result, err := translator.Translate("subAccountIN[s1,s2,s3];clusterIdIN[c1,c2];globalAccountId=5")
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("(key='subAccount' AND value IN (s1,s2,s3)) AND (key='clusterId' AND value IN (c1,c2)) AND (key='globalAccountId' AND value = 5)"))
			})
		})

		Context("Called with multivalue operator and single value", func() {
			It("Should return proper result surrounded in brackets", func() {
				result, err := translator.Translate("subAccountIN[s1]")
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("(key='subAccount' AND value IN (s1))"))
			})
		})

		Context("Called with missing operator", func() {
			It("Should return error", func() {
				_, err := translator.Translate("subAccount[s1];clusterIdIN[c1,c2];globalAccountId=5")
				Expect(err).To(MatchError("label query operator is missing"))
			})
		})

		Context("Called with multiple values for single value operator", func() {
			It("Should return error", func() {
				_, err := translator.Translate("subAccount=[s1,s2,s3];clusterIdIN[c1,c2];globalAccountId=5")
				Expect(err).To(MatchError("multiple values received for single value operation"))
			})
		})
	})
})
