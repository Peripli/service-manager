package cascade_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
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

	JustBeforeEach(func() {
		initTenantResources(true)
	})

	Context("cleanup", func() {
		It("finished tree should be deleted", func() {
			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

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
			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)
			triggerCascadeOperation(context.Background(), types.PlatformType, platformID, "root1")
			triggerCascadeOperation(context.Background(), types.ServiceBrokerType, brokerID, "root2")

			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					query.ByField(query.InOperator, "cascade_root_id", rootOpID, "root1", "root2"))
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*20+pollCascade*20+cleanupInterval*2).Should(Equal(0))
		})

		It("in_progress tree should not be deleted", func() {
			registerBindingLastOPHandlers(brokerServer, http.StatusOK, types.IN_PROGRESS)
			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

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
