package test

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func DescribeDeleteTestsfor(ctx *common.TestContext, t TestCase) bool {
	return Describe(fmt.Sprintf("DELETE %s", t.API), func() {
		var testResource common.Object
		var testResourceID string

		Context("Existing resource", func() {
			BeforeEach(func() {
				createFunc := t.DELETE.ResourceCreationBlueprint

				By(fmt.Sprintf("[SETUP]: Creating test resource of type %s", t.API))
				testResource = createFunc(ctx)
				Expect(testResource).ToNot(BeEmpty())

				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an non empty id of type string", testResource))
				testResourceID = testResource["id"].(string)
				Expect(testResourceID).ToNot(BeEmpty())
			})

			It("returns 200", func() {
				By("[TEST]: Verify resource of type %s exists before delete")
				ctx.SMWithOAuth.GET(fmt.Sprintf("/v1/%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusOK).JSON().Object().ContainsMap(testResource)

				By("[TEST]: Verify resource of type %s is deleted successfully")
				ctx.SMWithOAuth.DELETE(fmt.Sprintf("/v1/%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusOK)

				By("[TEST]: Verify resource of type %s does not exist after delete")
				ctx.SMWithOAuth.GET(fmt.Sprintf("/v1/%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusNotFound)

				//TODO we need to create cascade resources though
				By("[TEST]: Verify resources marked for cascade deletion are deleted")
				for _, resource := range t.DELETE.CascadeDeletions {
					ctx.SMWithOAuth.GET("/v1/" + resource.Child).WithQueryString("fieldQuery=" + resource.ChildReference + "+=+" + testResourceID).
						Expect().
						Status(http.StatusOK).JSON().Object().Value(resource.Child).Array().Empty()
				}
			})
		})

		Context("Not existing resource", func() {
			BeforeEach(func() {
				testResourceID = "non-existing-id"
			})

			It("returns 404", func() {
				ctx.SMWithOAuth.DELETE(fmt.Sprintf("/v1/%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
			})
		})
	})
}
