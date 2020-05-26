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
						ID:        rootOp,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:   "bla",
					ResourceID:    tenantId,
					CascadeRootID: rootOp,
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
						ID:        rootOp,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:  "bla",
					ResourceID:   tenantId,
					State:        types.PENDING,
					Type:         types.DELETE,
					ResourceType: types.TenantType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				AssertOperationCount(func(count int) { Expect(count).To(Equal(1)) }, query.ByField(query.EqualsOperator, "resource_id", tenantId))
			})
		})

		Context("cascade ops", func() {
			It("should succeed", func() {
				op := types.Operation{
					Base: types.Base{
						ID:        rootOp,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:   "bla",
					CascadeRootID: rootOp,
					ResourceID:    tenantId,
					Type:          types.DELETE,
					ResourceType:  types.TenantType,
				}
				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				// direct root children
				AssertOperationCount(func(count int) { Expect(count).To(Equal(3)) }, query.ByField(query.EqualsOperator, "parent_id", rootOp))
				// whole tree
				AssertOperationCount(func(count int) { Expect(count).To(Equal(tenantOperationsCount)) }, queryForOperationsInTheSameTree)
			})
		})

	})
})
