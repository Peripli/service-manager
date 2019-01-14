package services2_test

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test"
)

func TestServiceOfferings(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Offerings Tests Suite")

}

var _ = test.DescribeTestsFor(test.TestCase{
	API:            "service_offerings",
	SupportsLabels: false,
	GET: &test.GET{
		ResourceBlueprint: blueprint,
	},
	LIST: &test.LIST{
		ResourceBlueprint:                      blueprint,
		ResourceWithoutNullableFieldsBlueprint: blueprint,
	},
},
)

func blueprint(ctx *common.TestContext) common.Object {

	_, cService, _ := common.GenerateTestServiceWithPlans()
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	so := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id = "+id).
		Expect().
		Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

	return so.Object().Raw()
}
