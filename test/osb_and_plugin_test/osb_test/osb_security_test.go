package osb_test

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo/v2"
	"github.com/tidwall/gjson"
)

var _ = Describe("OSB Security", func() {
	var planID, serviceID, brokerID string
	var origBrokerExpect *httpexpect.Expect

	createBinding := func() *httpexpect.Response {
		return origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/12345/service_bindings/5678", web.OSBURL, brokerID)).
			WithJSON(common.Object{
				"service_id": serviceID,
				"plan_id":    planID,
				"context": common.Object{
					"platform": "kubernetes",
				},
			}).Expect().Status(http.StatusCreated)
	}

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

		origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
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
		Context("bindings", func() {
			BeforeEach(func() {
				createBinding()
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

		Context("instances", func() {
			It("should respond with the instance", func() {
				origBrokerExpect.GET(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
					Expect().Status(http.StatusOK).JSON().Object().Empty()
			})
			It("should be successful", func() {
				origBrokerExpect.DELETE(fmt.Sprintf("%s/%s/v2/service_instances/12345", web.OSBURL, brokerID)).
					Expect().Status(http.StatusOK).JSON().Object().Empty()
			})
		})
	})

	Context("from other subaccount", func() {
		var SMPlatformOtherSubaccountExpect *httpexpect.Expect
		var brokerExpect *httpexpect.Expect
		BeforeEach(func() {
			otherTenantExpect := ctx.NewTenantExpect("sm", "other-tenant")

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
			createBinding()
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
