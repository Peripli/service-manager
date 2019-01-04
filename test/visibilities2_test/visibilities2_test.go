package visibilities2_test

import (
	"fmt"
	"github.com/Peripli/service-manager/test/common"
	"net/http"
	"testing"

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
	POST: &test.POST{
		Prerequisites:        nil,
		AcceptsID:            false,
		PostRequestBlueprint: nil,
	},
	GET: &test.GET{
		ResourceCreationBlueprint: blueprint,
	},
	LIST: &test.LIST{
		ResourceCreationBlueprint: blueprint,
	},
	PATCH:      &test.PATCH{},
	DELETE:     &test.DELETE{},
	DELETELIST: &test.DELETELIST{},
})

func blueprint(ctx *common.TestContext) common.Object {

	_, cPaidPlan, _ := common.GeneratePaidTestPlan()
	_, cService, _ := common.GenerateTestServiceWithPlans(cPaidPlan)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	so := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id+=+"+id).
	Expect().
	Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

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
	},
//test.TestCase{
//API: "visibilities",
//Prerequisites: {
//resource: []Prerequisite{
//{
//name: "platforms",
//field: "platform_id",
//required: func()bool {
//// required if there are other visibilities for that plan id
//},
//
//},
//{
//name: "service_plans",
//field: "service_plan_id",
//required: func()bool,
//
//},
//},
//},
//Op: []string{"list", "get"},
//RandomResourceObjectGenerator: func(ctx *common.TestContext) common.Object {
//
//_, cPaidPlan, _ := common.GeneratePaidTestPlan()
//_, cService, _ := common.GenerateTestServiceWithPlans(cPaidPlan)
//catalog := common.NewEmptySBCatalog()
//catalog.AddService(cService)
//id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)
//
//so := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id+=+"+id).
//Expect().
//Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()
//
//servicePlanID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("fieldQuery", fmt.Sprintf("service_offering_id+=+%s", so.Object().Value("id").String().Raw())).
//Expect().
//Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()
//
//platformID := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.GenerateRandomPlatform()).
//Expect().
//Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
//
//visibility := ctx.SMWithOAuth.POST("/v1/visibilities").WithJSON(common.Object{
//"platform_id":     platformID,
//"service_plan_id": servicePlanID,
//}).Expect().
//Status(http.StatusCreated).JSON().Object().Raw()
//
//return visibility
//},
