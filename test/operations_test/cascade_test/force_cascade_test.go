package cascade_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("cascade force delete", func() {
	Context("force delete tenant", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("should succeed: delete all resources without using force", func() {
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
			validateDuplicationsCopied(fullTree)

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.ResourceType == types.ServiceInstanceType && len(operation.Errors) > 2
			})).To(Equal(0))
		})
	})

	Context("force delete instances", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("should succeed: delete instance using force", func() {
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
			}, actionTimeout*7+pollCascade*7).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.SUCCEEDED
			})).To(Equal(tenantOperationsCount))

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.ResourceType == types.ServiceInstanceType && len(operation.Errors) > 2
			})).To(Equal(3))
		})

		It("should succeed: delete container using force", func() {
			// tree contains this branch: sa(root) -> platform -> instance(container) -> instance(in container) -> binding
			createContainerWithChildren()
			// binding under container will have failed state in operation, failure will propagate up through the container as well
			registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
			registerBindingLastOPHandlers(globalBrokerServer, http.StatusInternalServerError, types.FAILED)

			newCtx := context.WithValue(context.Background(), cascade.ParentInstanceLabelKey{}, "containerID")
			rootID := triggerCascadeOperation(newCtx, types.TenantType, tenantID, true)

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
			validateDuplicationsCopied(fullTree)

			By("validating tenant error is a collection of his child errors")

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.SUCCEEDED
			})).To(Equal(tenantOperationsCount + 5))

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.ResourceType == types.ServiceBindingType && len(operation.Errors) > 2
			})).To(Equal(3))
		})
	})

	Context("force delete binding", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("should succeed: delete binding using force", func() {
			registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
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
			validateDuplicationsCopied(fullTree)

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.SUCCEEDED
			})).To(Equal(tenantOperationsCount))

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.ResourceType == types.ServiceBindingType && len(operation.Errors) > 2
			})).To(Equal(2))
		})

		It("should fail: delete binding using force", func() {
			// need to check that errors include db err

			registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)
			createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding3", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*5+pollCascade*5).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.SUCCEEDED && operation.ResourceType == types.ServiceBindingType && len(operation.Errors) > 2
			})).To(Equal(2))

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.FAILED && operation.ResourceType == types.ServiceInstanceType && len(operation.Errors) > 2
			})).To(Equal(1))
		})

	})

	Context("force delete with direct instance children only", func() {
		JustBeforeEach(func() {
			initTenantResources(false)
			ctx.SMWithBasic.SetBasicCredentials(ctx, ctx.TestPlatform.Credentials.Basic.Username, ctx.TestPlatform.Credentials.Basic.Password)
			createOSBInstance(ctx, ctx.SMWithBasic, globalBrokerID, osbInstanceID, map[string]interface{}{
				"service_id":        "global-service",
				"plan_id":           "global-plan",
				"organization_guid": "my-orgafsf",
				"context": map[string]string{
					"tenant": tenantID,
				},
			})
		})

		It("should failed: delete instance without children", func() {
			globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
				return http.StatusBadRequest, common.Object{}
			})
			rootID := triggerCascadeOperation(context.Background(), types.ServiceInstanceType, osbInstanceID, true)
			createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding3", map[string]interface{}{
				"service_id":        "global-service",
				"plan_id":           "global-plan",
				"organization_guid": "my-org",
			})
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+pollCascade*2).Should(Equal(1))
		})

		It("should succeed: delete instance without children", func() {
			globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
				return http.StatusBadRequest, common.Object{}
			})
			rootID := triggerCascadeOperation(context.Background(), types.ServiceInstanceType, osbInstanceID, true)
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+pollCascade*2).Should(Equal(1))
		})

		It("should succeed: delete tenant with only direct instance children", func() {
			globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
				return http.StatusBadRequest, common.Object{}
			})
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
			}, actionTimeout*5+pollCascade*5).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.SUCCEEDED && operation.ResourceType == types.ServiceInstanceType && len(operation.Errors) > 2
			})).To(Equal(1))
		})

		It("should fail: delete tenant with only direct instance children", func() {
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)
			createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding3", map[string]interface{}{
				"service_id":        "global-service",
				"plan_id":           "global-plan",
				"organization_guid": "my-org",
			})

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*3+pollCascade*3).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.FAILED && operation.ResourceType == types.ServiceInstanceType && len(operation.Errors) > 2
			})).To(Equal(1))

			Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
				return operation.State == types.SUCCEEDED && len(operation.Errors) < 3
			})).To(Equal(2))
		})
	})
})
