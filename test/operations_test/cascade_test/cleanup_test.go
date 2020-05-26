package cascade_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Cleanup Finished Cascade Operations", func() {

	Context("Single Tree Cleanup", func() {
		It("Should succeed", func() {
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

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			ctx.Maintainer.CleanupFinishedCascadeOperations()
			tree, err = fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(tree.byOperationID)).To(Equal(0))
		})
	})
})
