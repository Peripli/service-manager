package cascade_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("cascade operations", func() {
	BeforeEach(func() {
		cleanupInterval = 100 * time.Millisecond
	})

	Context("cleanup", func() {
		AfterEach(func() {
			ctx.Cleanup()
		})

		It("should cleaned", func() {
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

			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11+cleanupInterval).Should(Equal(0))
		})
	})
})
