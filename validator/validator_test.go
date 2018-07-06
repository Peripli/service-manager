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

package validator

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator")
}

func assertHasReservedCharacters(input string) {
	It("should return true", func() {
		Expect(HasRFC3986ReservedSymbols(input)).To(Equal(true))
	})
}

func assertNoReservedCharacters(input string) {
	It("should return false", func() {
		Expect(HasRFC3986ReservedSymbols(input)).To(Equal(false))
	})
}

func assertReservedCases(cases []string, hasReserved bool) {
	for _, str := range cases {
		if hasReserved {
			assertHasReservedCharacters(str)
		} else {
			assertNoReservedCharacters(str)
		}
	}
}

var _ = Describe("Validator test", func() {
	Context("HasRFC3986ReservedSymbols with single characters", func() {
		reserved := []string{":", "/", "?", "#", "[", "]", "@", "!", "$", "&", "'", "(", ")", "*", "+", ",", ";", "="}
		assertReservedCases(reserved, true)
	})

	Context("HasRFC3986ReservedSymbols with multiple symbols", func() {
		cases := []string{"@a\\/", "@a@", "a:b", "a:;b", ":;@", "()", "+a+", "[a+]", "a=3?"}
		assertReservedCases(cases, true)
	})

	Context("HasRFC3986ReservedSymbols with no reserved symbols", func() {
		cases := []string{"a", "a~b", "a_b", "a-b", "", "74a", "a00", "--a", "-a", "a-", "a--", "-"}
		assertReservedCases(cases, false)
	})
})
