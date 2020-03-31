package osb_test

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	"github.com/tidwall/gjson"
)

var _ = Describe("OSB Security", func() {
	var planID, serviceID, brokerID string
	var origBrokerExpect *httpexpect.Expect

	BeforeEach(func() {
		plan := common.GenerateFreeTestPlan()
		planID = gjson.Get(plan, "id").String()
		service := common.GenerateTestServiceWithPlansWithID("1", plan)
		serviceID = gjson.Get(service, "id").String()

		catalog := common.NewEmptySBCatalog()
		catalog.AddService(service)

		brokerUtils := ctx.RegisterBrokerWithCatalog(catalog)
		brokerID = brokerUtils.Broker.ID

		platformForTenant := common.RegisterPlatformInSM(common.Object{
			"name": "test1",
			"type": "kubernetes",
		}, ctx.SMWithOAuthForTenant, nil)

		SMPlatformExpect := ctx.SM.Builder(func(req *httpexpect.Request) {
			username := platformForTenant.Credentials.Basic.Username
			password := platformForTenant.Credentials.Basic.Password
			req.WithBasicAuth(username, password)
		})

		SMPlatformExpect.PUT(web.BrokerPlatformCredentialsURL).
			WithJSON(common.Object{
				"broker_id":     brokerID,
				"username":      "admin",
				"password_hash": hashPassword("admin"),
			}).Expect().Status(http.StatusOK)

		common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, brokerID)

		origBrokerExpect = ctx.SM.Builder(func(req *httpexpect.Request) {
			req.WithBasicAuth("admin", "admin")
		})
		// '{"service_id":"33ceba5779bfa320a1ef0694d98069df", "plan_id":"a80bf06fbd20eff6a5b6896e873d8cbe","space_guid":"sdaf", "organization_guid":"gggfdgd","context":{"organization_guid":"blabla", "space_guid":"asdfgasdf", "platform":"cloudfoundry"}}'
		origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
			WithJSON(common.Object{
				"service_id": serviceID,
				"plan_id":    planID,
				"context": common.Object{
					"platform": "kubernetes",
				},
			}).Expect().Status(http.StatusCreated)

		// {"service_id":"33ceba5779bfa320a1ef0694d98069df", "plan_id":"a80bf06fbd20eff6a5b6896e873d8cbe","space_guid":"sdaf", "organization_guid":"gggfdgd","context":{"organization_guid":"blabla", "space_guid":"asdfgasdf", "platform":"cloudfoundry"}}
		origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/12345/service_bindings/5678", web.OSBURL, brokerID)).
			WithJSON(common.Object{
				"service_id": serviceID,
				"plan_id":    planID,
				"context": common.Object{
					"platform": "kubernetes",
				},
			}).Expect().Status(http.StatusCreated)
	})

	AfterEach(func() {
		ctx.CleanupBroker(brokerID)
		ctx.CleanupPlatforms()
	})

	Context("from the same subaccount", func() {
		Context("get instance", func() {
			It("should respond with the instance", func() {
				origBrokerExpect.GET(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
					Expect().Status(http.StatusOK).JSON().Object().Empty()
			})
		})

		Context("delete instance", func() {
			It("should be successful", func() {
				origBrokerExpect.DELETE(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
					Expect().Status(http.StatusOK).JSON().Object().Empty()
			})
		})

		Context("get binding", func() {
			It("should respond with the binding", func() {
				origBrokerExpect.GET(fmt.Sprintf("%s/%s/v2/service_instances/12345/service_bindings/5678", web.OSBURL, brokerID)).
					Expect().Status(http.StatusOK).JSON().Object().ContainsKey("credentials")
			})
		})

		Context("delete binding", func() {
			It("should be successful", func() {
				origBrokerExpect.DELETE(fmt.Sprintf("%s/%s/v2/service_instances/12345/service_bindings/5678", web.OSBURL, brokerID)).
					Expect().Status(http.StatusOK).JSON().Object().Empty()
			})
		})
	})

	Context("from other subaccount", func() {
		var SMPlatformOtherSubaccountExpect *httpexpect.Expect
		var brokerExpect *httpexpect.Expect
		BeforeEach(func() {
			otherTenantExpect := ctx.NewTenantExpect("sm", "other-tenant",
				"service_manager.subaccount.broker.read",
				"service_manager.subaccount.broker.manage",
				"service_manager.subaccount.platform.read",
				"service_manager.subaccount.platform.manage",
				"service_manager.subaccount.platform.manage",
				"service_manager.subaccount.service_plan.read",
				"service_manager.subaccount.service_offering.read",
				"service_manager.subaccount.service_instance.read",
				"service_manager.subaccount.service_instance.manage",
				"service_manager.subaccount.service_binding.read",
				"service_manager.subaccount.service_binding.manage",
				"service_manager.subaccount.service_plan.read",
				"service_manager.subaccount.service_offering.read",
			)

			platformForOtherTenant := common.RegisterPlatformInSM(common.Object{
				"name": "test2",
				"type": "kubernetes",
			}, otherTenantExpect, nil)

			SMPlatformOtherSubaccountExpect = ctx.SM.Builder(func(req *httpexpect.Request) {
				username := platformForOtherTenant.Credentials.Basic.Username
				password := platformForOtherTenant.Credentials.Basic.Password
				req = req.WithBasicAuth(username, password)
			})

			SMPlatformOtherSubaccountExpect.PUT(web.BrokerPlatformCredentialsURL).
				WithJSON(common.Object{
					"broker_id":     brokerID,
					"username":      "admin2",
					"password_hash": hashPassword("admin"),
				}).Expect().Status(http.StatusOK)

			brokerExpect = ctx.SM.Builder(func(req *httpexpect.Request) {
				req = req.WithBasicAuth("admin2", "admin")
			})
		})

		Context("get instance", func() {
			It("should return not found", func() {
				brokerExpect.GET(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
					Expect().Status(http.StatusNotFound).JSON().Object().ValueEqual("description", "could not find such service instance")
			})
		})

		Context("delete instance", func() {
			It("should return not found", func() {
				brokerExpect.DELETE(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
					Expect().Status(http.StatusNotFound).JSON().Object().ValueEqual("description", "could not find such service instance")
			})
		})

		Context("get binding", func() {
			It("should return not found", func() {
				brokerExpect.GET(fmt.Sprintf("%s/%s/v2/service_instances/12345/service_bindings/5678", web.OSBURL, brokerID)).
					Expect().Status(http.StatusNotFound).JSON().Object().ValueEqual("description", "could not find such service instance")
			})
		})

		Context("delete binding", func() {
			It("should return not found", func() {
				brokerExpect.DELETE(fmt.Sprintf("%s/%s/v2/service_instances/12345/service_bindings/5678", web.OSBURL, brokerID)).
					Expect().Status(http.StatusNotFound).JSON().Object().ValueEqual("description", "could not find such service instance")
			})
		})
	})
})
