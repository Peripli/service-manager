package context_signature

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	"net/http"
)

var _ = Describe("context signature verification tests", func() {

	AfterEach(func() {
		brokerServer.ResetHandlers()
		common.RemoveAllBindings(ctx)
		common.RemoveAllInstances(ctx)
	})

	FContext("OSB", func() {
		instanceID := "signed-ctx-instance"
		var provisionFunc func() string
		BeforeEach(func() {
			provisionFunc = common.GetOsbProvisionFunc(ctx, instanceID, osbURL, catalogServiceID, catalogPlanID)
		})
		When("provisioning a service instance", func() {
			It("should have a valid context signature on the request body", func() {
				common.ProvisionInstanceAndVerifySignature(ctx, brokerServer, provisionFunc, publicKeyStr)
			})
		})
		When("updating a service instance", func() {
			It("should have a valid context signature on the request body", func() {
				common.ProvisionInstanceAndVerifySignature(ctx, brokerServer, provisionFunc, publicKeyStr)
				ctx.SMWithBasic.PATCH(osbURL + "/v2/service_instances/" + instanceID).
					WithJSON(common.JSONToMap(fmt.Sprintf(common.CFContext, catalogServiceID, catalogPlanID, "updated-instance-name"))).
					Expect().
					Status(http.StatusOK)
				common.VerifySignatureNotPersisted(ctx, types.ServiceInstanceType, instanceID)
			})
		})
		When("binding a service instance", func() {
			It("should have a context signature on the request body", func() {
				common.ProvisionInstanceAndVerifySignature(ctx, brokerServer, provisionFunc, publicKeyStr)

				brokerServer.BindingHandler = common.GetVerifyContextHandlerFunc(publicKeyStr)
				bindingID := "signed-ctx-instance-binding-id"
				common.OsbBind(ctx, instanceID, bindingID, osbURL, catalogServiceID, catalogPlanID)

				common.VerifySignatureNotPersisted(ctx, types.ServiceBindingType, bindingID)
			})
		})
		When("sm environment variable context_rsa_public_key is set", func() {
			It("should return it in /info API", func() {
				ctx.SM.GET(web.InfoURL).Expect().
					Status(http.StatusOK).
					JSON().Object().Value("context_rsa_public_key").Equal(publicKeyStr)
			})
		})
	})

	Context("SMAAP", func() {
		var provisionFunc func() string
		BeforeEach(func() {
			provisionFunc = common.GetSMAAPProvisionInstanceFunc(ctx, "false", planID)
		})
		When("provisioning a service instance", func() {
			It("should have a valid context signature on the request body", func() {
				common.ProvisionInstanceAndVerifySignature(ctx, brokerServer, provisionFunc, publicKeyStr)
			})
		})
		When("updating a service instance", func() {
			It("should have a valid context signature on the request body", func() {
				instanceID := common.ProvisionInstanceAndVerifySignature(ctx, brokerServer, provisionFunc, publicKeyStr)
				patchRequestBody := common.Object{
					"name": "updated-test-instance",
				}
				ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
					WithJSON(patchRequestBody).
					Expect().
					Status(http.StatusOK)

				common.VerifySignatureNotPersisted(ctx, types.ServiceInstanceType, instanceID)

				ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID).
					Expect().Status(http.StatusOK).
					JSON().
					Object().Value("context").Object().NotContainsKey("signature")
			})
		})
		When("binding a service instance", func() {
			It("should have a context signature on the request body", func() {
				instanceID := common.ProvisionInstanceAndVerifySignature(ctx, brokerServer, provisionFunc, publicKeyStr)

				brokerServer.BindingHandler = common.GetVerifyContextHandlerFunc(publicKeyStr)

				bindingID := common.SmaapBind(ctx, "false", instanceID)

				common.VerifySignatureNotPersisted(ctx, types.ServiceBindingType, bindingID)
			})
		})
	})
})
