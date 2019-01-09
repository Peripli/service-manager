package testy_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTesty(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testy Tests Suite")

}

func resourceCreationBlueprint(ctx *common.TestContext) common.Object {

	_, cPaidPlan, _ := common.GeneratePaidTestPlan()
	_, cService, _ := common.GenerateTestServiceWithPlans(cPaidPlan)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	object := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id+=+"+id).
		Expect()

	so := object.Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

	servicePlanID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("fieldQuery", fmt.Sprintf("service_offering_id+=+%s", so.Object().Value("id").String().Raw())).
		Expect().
		Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()

	platformID := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.GenerateRandomPlatform()).
		Expect().
		Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

	visibility := ctx.SMWithOAuth.POST("/v1/visibilities").WithJSON(common.Object{
		"platform_id":     platformID,
		"service_plan_id": servicePlanID,
	}).Expect().
		Status(http.StatusCreated).JSON().Object().Raw()

	return visibility
}

var _ = Describe("Bar", func() {
	Describe("d1", func() {
		ctx := common.NewTestContext(nil)
		func() {
			defer GinkgoRecover()
			e := recover()
			if e != nil {
				panic(e)
			}
			resourceCreationBlueprint(ctx)

		}()
		BeforeSuite(func() {
			fmt.Println("hi")
		})

		BeforeEach(func() {
			fmt.Println("yo")
		})
		It("asd", func() {
			fmt.Sprintf("hi")
		})
	})

})
