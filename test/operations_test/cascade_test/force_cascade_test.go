package cascade_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var createInstances = true

var _ = Describe("force cascade delete", func() {
	JustBeforeEach(func() {
		initTenantResources(createInstances)
	})

	Context("tenant with multilevel subtree", func() {
		BeforeEach(func() {
			createInstances = true
		})

		Context("should succeed", func() {
			It("without errors", func() {
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
				validateCountOfOperationsThatDeletedUsingForce(fullTree, types.ServiceInstanceType, 0)
			})

			It("without errors and one of the instances is in orphan mitigation", func() {
				pollingCount := 0
				tenantBrokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
					if pollingCount == 0 {
						pollingCount++
						return http.StatusOK, common.Object{"state": "in progress"}
					} else {
						return http.StatusOK, common.Object{"state": "failed"}
					}
				})

				rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

				By("validating binding failed and marked as orphan mitigation")
				Eventually(func() int {
					count, err := ctx.SMRepository.Count(
						context.Background(),
						types.OperationType,
						queryForOperationsInTheSameTree(rootID),
						queryFailures,
						queryForOrphanMitigationOperations,
						queryForBindingsOperations)
					Expect(err).NotTo(HaveOccurred())

					return count
				}, actionTimeout*4+pollCascade*4).Should(Equal(2))

				By("validating that instances without bindings were deleted")
				Eventually(func() int {
					count, err := ctx.SMRepository.Count(
						context.Background(),
						types.OperationType,
						queryForOperationsInTheSameTree(rootID),
						querySucceeded,
						queryForInstanceOperations)
					Expect(err).NotTo(HaveOccurred())

					return count
				}, actionTimeout*2+pollCascade*2).Should(Equal(5))

				tenantBrokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
					return http.StatusOK, common.Object{"state": "succeeded"}
				})

				By("validating bindings released from orphan mitigation and mark as succeeded")
				Eventually(func() int {
					count, err := ctx.SMRepository.Count(
						context.Background(),
						types.OperationType,
						queryForBindingsOperations,
						queryForOperationsInTheSameTree(rootID),
						querySucceeded)
					Expect(err).NotTo(HaveOccurred())

					return count
				}, actionTimeout*2+maintainerRetry*2+cascadeOrphanMitigation*4).Should(Equal(4))

				By("validating root is succeeded")
				Eventually(func() int {
					count, err := ctx.SMRepository.Count(
						context.Background(),
						types.OperationType,
						queryForRoot(rootID),
						querySucceeded)
					Expect(err).NotTo(HaveOccurred())

					return count
				}, actionTimeout*8+maintainerRetry*8).Should(Equal(1))

				fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
				Expect(err).NotTo(HaveOccurred())

				validateParentsRanAfterChildren(fullTree)
				validateDuplicationsCopied(fullTree)
				validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			})

			It("with deprovision errors", func() {
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
				validateAllOperationsHasTheSameState(fullTree, types.SUCCEEDED, tenantOperationsCount)
				validateCountOfOperationsThatDeletedUsingForce(fullTree, types.ServiceInstanceType, 3)
			})

			It("with container deprovision errors", func() {
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
				validateAllOperationsHasTheSameState(fullTree, types.SUCCEEDED, tenantOperationsCount+5)
				validateCountOfOperationsThatDeletedUsingForce(fullTree, types.ServiceBindingType, 3)
			})

			It("with unbind errors", func() {
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
				validateAllOperationsHasTheSameState(fullTree, types.SUCCEEDED, tenantOperationsCount)
				validateCountOfOperationsThatDeletedUsingForce(fullTree, types.ServiceBindingType, 2)
			})
		})

		Context("should fail", func() {
			It("with database errors", func() {
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

				By("validating bindings were removed using force")
				Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
					return operation.State == types.SUCCEEDED && operation.ResourceType == types.ServiceBindingType && len(operation.Errors) > 2
				})).To(Equal(2))

				By("validating instance failed to removed using force")
				Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
					return operation.State == types.FAILED && operation.ResourceType == types.ServiceInstanceType && len(operation.Errors) > 2
				})).To(Equal(1))
			})
		})
	})

	Context("tenant with one level subtree", func() {
		BeforeEach(func() {
			createInstances = false
		})

		JustBeforeEach(func() {
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

		When("should succeed", func() {
			It("with deprovision errors", func() {
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

				By("validating instance removed using force")
				Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
					return operation.State == types.SUCCEEDED && operation.ResourceType == types.ServiceInstanceType && len(operation.Errors) > 2
				})).To(Equal(1))
			})
		})

		When("should fail", func() {
			It("with database errors", func() {
				globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
					return http.StatusBadRequest, common.Object{}
				})
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

	It("tenant with no children", func() {
		rootID := triggerCascadeOperation(context.Background(), types.TenantType, "some-tenant", true)

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
	})

	Context("instance", func() {
		BeforeEach(func() {
			createInstances = false
		})

		JustBeforeEach(func() {
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

		When("should fail", func() {
			It("with database errors", func() {
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
		})

		When("should succeed", func() {
			It("with deprovision errors", func() {
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
		})
	})
})

func validateCountOfOperationsThatDeletedUsingForce(fullTree *tree, objectType types.ObjectType, countOfForcedOperations int) {
	By(fmt.Sprintf("validating there are %v instances that removed using force", countOfForcedOperations))
	Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
		return operation.ResourceType == objectType && operationHasErrors(operation)
	})).To(Equal(countOfForcedOperations))
}

func operationHasErrors(operation *types.Operation) bool {
	return len(operation.Errors) > 2
}
