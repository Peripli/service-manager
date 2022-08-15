package plugin_test

import (
	"context"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/test/common"
	"net/http"
)

var _ = Describe("Platform Suspension OSB plugin", func() {
	var (
		ctx       *common.TestContext
		planID    string
		serviceID string
		brokerID  string
		osbURL    string
	)

	BeforeEach(func() {
		ctx = common.NewTestContextBuilderWithSecurity().Build()
		UUID, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		planID = UUID.String()
		plan1 := common.GenerateTestPlanWithID(planID)
		UUID, err = uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		serviceID = UUID.String()
		service1 := common.GenerateTestServiceWithPlansWithID(serviceID, plan1)
		catalog := common.NewEmptySBCatalog()
		catalog.AddService(service1)

		brokerID, _, _ = ctx.RegisterBrokerWithCatalog(catalog).GetBrokerAsParams()
		common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, brokerID)
		osbURL = "/v1/osb/" + brokerID

		ctx.SMWithBasic.PUT(osbURL+"/v2/service_instances/12345").
			WithHeader("Content-Type", "application/json").
			WithJSON(object{"service_id": serviceID, "plan_id": planID}).
			Expect().Status(http.StatusCreated)

		testPlatformID := ctx.TestPlatform.GetID()

		err = ctx.SMRepository.InTransaction(context.TODO(), func(ctx context.Context, storage storage.Repository) error {
			var updatedPlatform types.Object
			byID := query.ByField(query.EqualsOperator, "id", testPlatformID)
			platformFromStorage, err := storage.Get(ctx, types.PlatformType, byID)
			Expect(err).ToNot(HaveOccurred())

			platformFromStorage.(*types.Platform).Suspended = true
			if updatedPlatform, err = storage.Update(ctx, platformFromStorage, types.LabelChanges{}); err != nil {
				return err
			}
			Expect(updatedPlatform.(*types.Platform).Suspended).To(Equal(true))
			return nil
		})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if ctx != nil {
			ctx.Cleanup()
		}
	})

	When("Platform Suspended", func() {

		It("should not allow provision request", func() {
			res := ctx.SMWithBasic.PUT(osbURL+"/v2/service_instances/12345").
				WithHeader("Content-Type", "application/json").
				WithJSON(object{"service_id": serviceID, "plan_id": planID}).
				Expect().Status(http.StatusBadRequest).JSON().Object()
			res.Value("description").Equal("platform suspended")
		})

		It("should not allow update request", func() {
			res := ctx.SMWithBasic.PATCH(osbURL+"/v2/service_instances/12345").
				WithHeader("Content-Type", "application/json").
				WithJSON(object{"service_id": serviceID, "plan_id": planID, "parameters": "{\"key\":\"val\"}"}).
				Expect().Status(http.StatusBadRequest).JSON().Object()
			res.Value("description").Equal("platform suspended")
		})

		It("should allow deprovision request", func() {
			ctx.SMWithBasic.DELETE(osbURL + "/v2/service_instances/12345").
				Expect().Status(http.StatusOK).JSON().Object()
		})

		It("should allow fetch instance", func() {
			ctx.SMWithBasic.GET(osbURL + "/v2/service_instances/12345").
				Expect().Status(http.StatusOK)
		})

		It("should not allow bind request", func() {
			res := ctx.SMWithBasic.PUT(osbURL + "/v2/service_instances/12345/service_bindings/5678").
				WithJSON(object{"service_id": serviceID, "plan_id": planID}).
				Expect().Status(http.StatusBadRequest).JSON().Object()
			res.Value("description").Equal("platform suspended")
		})

		It("should allow unbind request", func() {
			ctx.SMWithBasic.DELETE(osbURL + "/v2/service_instances/12345/service_bindings/5678").
				Expect().Status(http.StatusOK)
		})

		It("should allow fetch binding request", func() {
			ctx.SMWithBasic.GET(osbURL + "/v2/service_instances/12345/service_bindings/5678").
				Expect().Status(http.StatusOK)
		})

		It("should allow get catalog request", func() {
			ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
				Expect().Status(http.StatusOK)
		})
	})
})
