package cascade_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("cascade force delete", func() {
	JustBeforeEach(func() {
		initTenantResources(true)
	})

	Context("should succeeded", func() {
		It("delete binding using force", func() {
			registerBindingLastOPHandlers(brokerServer, http.StatusInternalServerError, types.FAILED)
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*3+pollCascade*3).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(2))

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Equal(osbInstanceID))
				Expect(e.ResourceID).To(Or(Equal("binding1"), Equal("binding2")))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Equal(types.ServiceBindingType))
				Expect(e.ParentType).To(Equal(types.ServiceInstanceType))
			}
		})

		It("delete instance using force", func() {
			globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
				return http.StatusAccepted, common.Object{"async": true}
			})
			registerInstanceLastOPHandlers(globalBrokerServer, http.StatusInternalServerError, types.FAILED)
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*3+pollCascade*3).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(3))

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Or(Equal(platformID), Equal(brokerID), Equal(tenantID)))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Equal(types.ServiceInstanceType))
				Expect(e.ParentType).To(Or(Equal(types.ServiceBrokerType), Equal(types.PlatformType), Equal(types.TenantType)))
			}
		})

		It("delete container using force", func() {

		})

		It("delete with only direct instance children", func() {

		})
	})

	Context("should failed", func() {
		It("failed to force delete binding", func() {
			// need to check that errors include db err
		})

		It("failed to delete with only direct instance children", func() {

		})
	})
})

func validateResourcesDeleted(repository storage.TransactionalRepository, byResourceType map[types.ObjectType][]*types.Operation) {
	By("validating resources have deleted")
	for objectType, operations := range byResourceType {
		if objectType != types.TenantType {
			IDs := make([]string, 0, len(operations))
			for _, operation := range operations {
				IDs = append(IDs, operation.ResourceID)
			}

			count, err := repository.Count(context.Background(), objectType, query.ByField(query.InOperator, "id", IDs...))
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(0), fmt.Sprintf("resources from type %s failed to be deleted", objectType))
		}
	}
}
