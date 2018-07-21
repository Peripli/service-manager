package util

import "testing"
import . "github.com/onsi/ginkgo"
import . "github.com/onsi/gomega"

func TestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Util test suite")
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
