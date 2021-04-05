package cascade_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("cascade operations", func() {
	Context("parallel delete", func() {

		BeforeEach(func() {
			initTenantResources(true, false)
		})

		When("Cascade deleting platform during tenant removal", func() {

			var platformOperationID string

			BeforeEach(func() {
				triggerCascadeOperation(context.Background(), types.TenantType, tenantID, false)

				platformOperation, err := ctx.SMRepository.Get(
					context.Background(),
					types.OperationType,
					query.ByField(query.EqualsOperator, "resource_id", platformID),
					query.ByField(query.EqualsOperator, "type", string(types.DELETE)))

				Expect(err).NotTo(HaveOccurred())

				platformOperationID = platformOperation.GetID()
			})

			It("Should get location to delete operation created by tenant deletion", func() {

				platformDeleteResponse := ctx.SMWithOAuth.DELETE(web.PlatformsURL+"/"+platformID).
					WithQuery("cascade", "true").
					Expect().
					Status(http.StatusAccepted)

				Expect(platformDeleteResponse.Header("Location").Raw()).To(HaveSuffix(platformOperationID))

			})

		})

		When("Deleting tenant during platform deletion", func() {

			BeforeEach(func() {

				ctx.SMWithOAuth.DELETE(web.PlatformsURL+"/"+platformID).
					WithQuery("cascade", "true").
					Expect().
					Status(http.StatusAccepted)

				triggerCascadeOperation(context.Background(), types.TenantType, tenantID, false)
			})

			It("should pass", func() {
				common.VerifyOperationExists(ctx, "", common.OperationExpectations{
					Category:          types.DELETE,
					State:             types.SUCCEEDED,
					ResourceType:      types.TenantType,
					Reschedulable:     false,
					DeletionScheduled: false,
				})
			})
		})

	})
})
