package plugin_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Service Manager Platform Termination Plugin Tests", func() {
	var (
		ctx          *common.TestContext
		brokerServer *common.BrokerServer
		osbURL       string

		brokerID  string
		serviceID string
		planID    string
	)

	JustBeforeEach(func() {
		username, password := test.RegisterBrokerPlatformCredentials(ctx.SMWithBasic, brokerID)
		ctx.SMWithBasic.SetBasicCredentials(ctx, username, password)
	})

	AfterEach(func() {
		if ctx != nil {
			ctx.Cleanup()
		}
	})

	Describe("Platform termination OSB plugin", func() {

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

			brokerID, _, brokerServer = ctx.RegisterBrokerWithCatalog(catalog).GetBrokerAsParams()
			brokerServer.ShouldRecordRequests(true)
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

				platformFromStorage.(*types.Platform).Active = false
				if updatedPlatform, err = storage.Update(ctx, platformFromStorage, types.LabelChanges{}); err != nil {
					return err
				}
				Expect(updatedPlatform.(*types.Platform).Active).To(Equal(false))
				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			ctx.SMWithOAuth.DELETE(web.PlatformsURL+"/"+testPlatformID).
				WithQuery("cascade", "true").
				Expect().
				Status(http.StatusAccepted)
		})

		osbOperations := []struct {
			name           string
			method         string
			path           string
			queries        []string
			expectedStatus int
		}{
			{"fetchCatalog", "GET", "/v2/catalog", []string{""}, http.StatusOK},
			{"provision", "PUT", "/v2/service_instances/1234", []string{""}, http.StatusUnprocessableEntity},
			{"provisionAsync", "PUT", "/v2/service_instances/1234", []string{"accepts_incomplete=true"}, http.StatusUnprocessableEntity},
			{"deprovision", "DELETE", "/v2/service_instances/1234", []string{""}, http.StatusOK},
			{"updateService", "PATCH", "/v2/service_instances/1234", []string{""}, http.StatusUnprocessableEntity},
			{"fetchService", "GET", "/v2/service_instances/1234", []string{""}, http.StatusOK},
			{"bind", "PUT", "/v2/service_instances/12345/service_bindings/111", []string{""}, http.StatusUnprocessableEntity},
			{"unbind", "DELETE", "/v2/service_instances/1234/service_bindings/111", []string{""}, http.StatusOK},
			{"fetchBinding", "GET", "/v2/service_instances/1234/service_bindings/111", []string{""}, http.StatusOK},
			//{"pollInstance", "GET", "/v2/service_instances/1234/last_operation", []string{"", "service_id=serviceId", "plan_id=planId", "operation=provision", "service_id=serviceId&plan_id=planId&operation=provision"}, http.StatusOK},
			//{"pollBinding", "GET", "/v2/service_instances/1234/service_bindings/111/last_operation", []string{"", "service_id=serviceId", "plan_id=planId", "operation=provision", "service_id=serviceId&plan_id=planId&operation=provision"}, http.StatusOK},
			{"adaptCredentials", "POST", "/v2/service_instances/1234/service_bindings/111/adapt_credentials", []string{""}, http.StatusOK},
		}

		for _, op := range osbOperations {
			op := op
			It(fmt.Sprintf("Plugin intercepts %s operation", op.name), func() {

				for _, query := range op.queries {
					ctx.SMWithBasic.Request(op.method, osbURL+op.path).
						WithHeader("Content-Type", "application/json").
						WithJSON(object{"service_id": serviceID, "plan_id": planID}).
						WithQueryString(query).
						Expect().
						Status(op.expectedStatus)
				}
			})
		}

	})

})
