package visibilities2_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test"
)

func TestVisibilities(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Visibilities Tests Suite")

}

var _ = test.DescribeTestsFor(test.TestCase{
	API:            "visibilities",
	SupportsLabels: true,
	//GET: &test.GET{
	//	ResourceBlueprint: blueprint(true),
	//},
	//LIST: &test.LIST{
	//	ResourceBlueprint:                      blueprint(true),
	//	ResourceWithoutNullableFieldsBlueprint: blueprint(false),
	//},
	//DELETE: &test.DELETE{
	//	ResourceCreationBlueprint: blueprint(true),
	//},
	DELETELIST: &test.DELETELIST{
		ResourceBlueprint:                      blueprint(true),
		ResourceWithoutNullableFieldsBlueprint: blueprint(false),
	},
})

func blueprint(withNullableFields bool) func(ctx *common.TestContext) common.Object {
	return func(ctx *common.TestContext) common.Object {
		visReqBody := make(common.Object, 0)
		_, cPaidPlan, _ := common.GeneratePaidTestPlan()
		_, cService, _ := common.GenerateTestServiceWithPlans(cPaidPlan)
		catalog := common.NewEmptySBCatalog()
		catalog.AddService(cService)
		id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

		object := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id = "+id).
			Expect()

		so := object.Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

		servicePlanID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("fieldQuery", fmt.Sprintf("service_offering_id = %s", so.Object().Value("id").String().Raw())).
			Expect().
			Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()
		visReqBody["service_plan_id"] = servicePlanID
		if withNullableFields {
			platformID := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.GenerateRandomPlatform()).
				Expect().
				Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
			visReqBody["platform_id"] = platformID
		}

		visibility := ctx.SMWithOAuth.POST("/v1/visibilities").WithJSON(visReqBody).Expect().
			Status(http.StatusCreated).JSON().Object().Raw()
		return visibility
	}
}
