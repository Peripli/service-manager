package plans2_test

import (
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
)

func TestServicePlans(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Plans Tests Suite")

}

var _ = test.DescribeTestsFor(test.TestCase{
	API: "service_plans",
	//Prerequisites: {
	//	[]resource: {
	//		name: "service_offerings",
	//		reference: "service_offerings_id",
	//		required: true,
	//	},
	//},
	Op: []string{"list", "get"},
	RandomResourceObjectGenerator: func(ctx *common.TestContext) common.Object {

		_, cPaidPlan, _ := common.GeneratePaidTestPlan()
		_, cService, _ := common.GenerateTestServiceWithPlans(cPaidPlan)
		catalog := common.NewEmptySBCatalog()
		catalog.AddService(cService)
		id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

		so := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id+=+"+id).
			Expect().
			Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

		sp := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("fieldQuery", fmt.Sprintf("service_offering_id+=+%s", so.Object().Value("id").String().Raw())).
			Expect().
			Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First()

		return sp.Object().Raw()
	},
},
)
