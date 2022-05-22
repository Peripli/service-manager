package context_signature

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("context signature verification tests", func() {

	AfterEach(func() {
		brokerServer.ResetHandlers()
		err := common.RemoveAllBindings(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		err = common.RemoveAllInstances(ctx)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("OSB", func() {
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
				object := ctx.SM.GET(web.InfoURL).Expect().
					Status(http.StatusOK).
					JSON().Object()
				object.Value("context_rsa_public_key").Equal(publicKeyStr)
				object.Value("context_successor_rsa_public_key").Equal(publicSuccessorKeyStr)
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

	Context("when the private key is invalid", func() {
		instanceID := "signed-ctx-instance-invalid"
		var provisionFunc func() string
		BeforeEach(func() {
			ctx = common.NewTestContextBuilderWithSecurity().WithEnvPostExtensions(func(e env.Environment, servers map[string]common.FakeServer) {
				e.Set("api.osb_rsa_private_key", "invalidKey")
			}).Build()
			registerServiceBroker()
		})
		It("should not fail the request,osb request sent without the signature", func() {
			provisionFunc = common.GetOsbProvisionFunc(ctx, instanceID, osbURL, catalogServiceID, catalogPlanID)
			common.ProvisionInstanceWithoutSignature(ctx, brokerServer, provisionFunc)
		})
		It("should not fail the request,SMAAP request sent without the signature", func() {
			provisionFunc = common.GetSMAAPProvisionInstanceFunc(ctx, "false", planID)
			common.ProvisionInstanceWithoutSignature(ctx, brokerServer, provisionFunc)
		})

	})

})
