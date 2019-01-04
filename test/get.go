package test

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func DescribeGetTestsfor(ctx *common.TestContext, t TestCase, r []common.Object) bool {
	return Describe("GET", func() {
		var testResource common.Object
		var testResourceID string

		Context(fmt.Sprintf("Existing resource of type %s", t.API), func() {
			BeforeEach(func() {
				testResource = r[0]
				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v is not empty", testResource))
				Expect(testResource).ToNot(BeEmpty())

				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an id of type string", testResource))
				testResourceID = testResource["id"].(string)
				Expect(testResourceID).ToNot(BeEmpty())
			})

			It("returns 200", func() {
				ctx.SMWithOAuth.GET(fmt.Sprintf("/v1/%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
			})
		})

		Context(fmt.Sprintf("Not existing resource of type %s", t.API), func() {
			BeforeEach(func() {
				testResourceID = "non-existing-id"
			})

			It("returns 404", func() {
				ctx.SMWithOAuth.GET(fmt.Sprintf("/v1/%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
			})
		})
	})
}
