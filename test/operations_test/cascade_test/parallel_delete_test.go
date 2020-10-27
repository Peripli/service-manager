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

		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("Cascade deleting platform during tenant removal should get location to delete operation created by tenant deletion", func() {

			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, false)

			platformOperation, err := ctx.SMRepository.Get(
				context.Background(),
				types.OperationType,
				query.ByField(query.EqualsOperator, "resource_id", platformID),
				query.ByField(query.EqualsOperator, "type", string(types.DELETE)))

			Expect(err).NotTo(HaveOccurred())
			platformOperationId := platformOperation.GetID()

			platformDeleteResponse := ctx.SMWithOAuth.DELETE(web.PlatformsURL+"/"+platformID).
				WithQuery("cascade", "true").
				Expect().
				Status(http.StatusAccepted)

			Expect(platformDeleteResponse.Header("Location").Raw()).To(HaveSuffix(platformOperationId))

		})

		It("Deleting tenant during platform deletion should pass", func() {

			ctx.SMWithOAuth.DELETE(web.PlatformsURL+"/"+platformID).
				WithQuery("cascade", "true").
				Expect().
				Status(http.StatusAccepted)

			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, false)
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
