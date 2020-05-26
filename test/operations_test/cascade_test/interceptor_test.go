package cascade_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Cascade Operation Interceptor", func() {

	AfterEach(func() {
		ctx.Cleanup()
	})
	Context("Create", func() {

		Context("having global instances", func() {
			It("should fail ", func() {
				common.CreateInstanceInPlatformForPlan(ctx, ctx.TestPlatform.ID, plan.GetID())
				op := types.Operation{
					Base: types.Base{
						ID:        rootOpID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:   "bla",
					ResourceID:    tenantID,
					CascadeRootID: rootOpID,
					State:         types.PENDING,
					Type:          types.DELETE,
					ResourceType:  types.TenantType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).To(HaveOccurred())
				expectedErrMsg := fmt.Sprintf("broker %s has instances from global platform", brokerID)
				Expect(err.Error()).To(Equal(expectedErrMsg))
			})
		})

		Context ("not cascade ops", func() {
			It("should skip", func() {
				op := types.Operation{
					Base: types.Base{
						ID:        rootOpID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:  "bla",
					ResourceID:   tenantID,
					State:        types.PENDING,
					Type:         types.DELETE,
					ResourceType: types.TenantType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				AssertOperationCount(func(count int) { Expect(count).To(Equal(1)) }, query.ByField(query.EqualsOperator, "resource_id", tenantID))
			})
		})

		Context("cascade ops", func() {
			It("should succeed", func() {
				op := types.Operation{
					Base: types.Base{
						ID:        rootOpID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:   "bla",
					CascadeRootID: rootOpID,
					ResourceID:    tenantID,
					Type:          types.DELETE,
					ResourceType:  types.TenantType,
				}
				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				tree, err := fetchFullTree(ctx.SMRepository, rootOpID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tree.byOperationID)).To(Equal(11))

				platformOpID := tree.byResourceID[platformID][0].ID
				brokerOpID := tree.byResourceID[brokerID][0].ID
				instanceOpID := tree.byParentID[platformOpID][0].ID

				// Tenant[broker, platform , smaap_instance]
				Expect(len(tree.byParentID[rootOpID])).To(Equal(3))
				// Platform [instance]
				Expect(len(tree.byParentID[platformOpID])).To(Equal(1))
				// Broker[instance, smaap_instance]
				Expect(len(tree.byParentID[brokerOpID])).To(Equal(2))
				// Instance[binding1, binding2]
				Expect(len(tree.byParentID[instanceOpID])).To(Equal(2))

			})
		})

	})
})
