package cascade_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
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
			triggerCascadeOperation(context.Background(), types.TenantType, tenantID)

			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11+cleanupInterval).Should(Equal(0))
		})

		It("multiple finished trees should be deleted", func() {

		})

		It("in_progress should not be deleted", func() {
			registerBindingLastOPHandlers(brokerServer, http.StatusOK, types.IN_PROGRESS)
			triggerCascadeOperation(context.Background(), types.TenantType, tenantID)

			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11+cleanupInterval).Should(Equal(11))
		})
	})
})
