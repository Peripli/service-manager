package cascade_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"strings"
	"time"
)

var createInstances bool

var _ = Describe("force cascade delete", func() {
	JustBeforeEach(func() {
		initTenantResources(createInstances)
	})

	waitCascadingProcessToFinish := func(timeout time.Duration, expectedTaskCyclesCount int, expectedOperationsCount int, operationQuery ...query.Criterion) {
		By("waiting cascading process to finish")
		Eventually(func() int {
			count, err := ctx.SMRepository.Count(
				context.Background(),
				types.OperationType,
				operationQuery...)
			Expect(err).NotTo(HaveOccurred())
			return count
		}, timeout*time.Duration(expectedTaskCyclesCount)).Should(Equal(expectedOperationsCount))
	}

	fetchTree := func(rootID string) *tree {
		fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
		Expect(err).ToNot(HaveOccurred())
		return fullTree
	}

	Context("delete tenant", func() {
		Context("tenant with multilevel subtree", func() {
			BeforeEach(func() {
				createInstances = true
			})

			When("there are no failures", func() {
				It("should marked as succeeded", func() {
					rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

					waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)

					fullTree := fetchTree(rootID)
					validateAllResourceDeleted(ctx.SMRepository, fullTree.byResourceType)
					validateParentsRanAfterChildren(fullTree)
					validateDuplicationHasTheSameState(fullTree)
					validateNumberOfForceDeletions(fullTree, types.ServiceInstanceType, types.SUCCEEDED, 0)
				})
			})

			When("one of the instances is in orphan mitigation", func() {
				It("should wait for the instance to be successfully deleted", func() {
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

					queryForBindingsInOrphanMitigation := []query.Criterion{queryForOperationsInTheSameTree(rootID), queryFailures, queryForOrphanMitigationOperations, queryForBindingsOperations}
					waitCascadingProcessToFinish(2*(actionTimeout+pollCascade), subaccountResources[types.ServiceBindingType], 2, queryForBindingsInOrphanMitigation...)

					queryForSucceededInstances := []query.Criterion{queryForOperationsInTheSameTree(rootID), querySucceeded, queryForInstanceOperations}
					waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResources[types.ServiceInstanceType], 5, queryForSucceededInstances...)

					tenantBrokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
						return http.StatusOK, common.Object{"state": "succeeded"}
					})

					queryForSucceededBindings := []query.Criterion{queryForBindingsOperations, queryForOperationsInTheSameTree(rootID), querySucceeded}
					waitCascadingProcessToFinish(actionTimeout+maintainerRetry+cascadeOrphanMitigation, subaccountResources[types.ServiceBindingType], 4, queryForSucceededBindings...)
					waitCascadingProcessToFinish(actionTimeout+maintainerRetry, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)

					fullTree := fetchTree(rootID)
					validateParentsRanAfterChildren(fullTree)
					validateDuplicationHasTheSameState(fullTree)
					validateAllResourceDeleted(ctx.SMRepository, fullTree.byResourceType)
				})
			})

			When("getting container's instance deprovision error", func() {
				It("should delete instance using force and marked as succeeded", func() {
					// tree contains this branch: sa(root) -> platform -> instance(container) -> instance(in container) -> binding
					createContainerWithChildren()
					// binding under container will have failed state in operation, failure will propagate up through the container as well
					registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
					registerBindingLastOPHandlers(globalBrokerServer, http.StatusInternalServerError, types.FAILED)

					newCtx := context.WithValue(context.Background(), cascade.ParentInstanceLabelKeys{}, []string{"containerID"})
					rootID := triggerCascadeOperation(newCtx, types.TenantType, tenantID, true)

					waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)

					fullTree := fetchTree(rootID)
					validateAllResourceDeleted(ctx.SMRepository, fullTree.byResourceType)
					validateParentsRanAfterChildren(fullTree)
					validateDuplicationHasTheSameState(fullTree)
					validateAllOperationsHasTheSameState(fullTree, types.SUCCEEDED, tenantOperationsCount+5)
					validateNumberOfForceDeletions(fullTree, types.ServiceBindingType, types.SUCCEEDED, 3)
				})
			})

			When("getting unbind error", func() {
				It("should delete binding using force and marked as succeeded", func() {
					registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
					rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

					waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForOperationsInTheSameTree(rootID), queryForRoot(rootID), querySucceeded)

					fullTree := fetchTree(rootID)
					validateAllResourceDeleted(ctx.SMRepository, fullTree.byResourceType)
					validateParentsRanAfterChildren(fullTree)
					validateDuplicationHasTheSameState(fullTree)
					validateAllOperationsHasTheSameState(fullTree, types.SUCCEEDED, tenantOperationsCount)
					validateNumberOfForceDeletions(fullTree, types.ServiceBindingType, types.SUCCEEDED, 2)
				})
			})

			When("getting deprovision error", func() {
				When("succeed to delete instance using force", func() {
					It("should marked as succeeded", func() {
						globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
							return http.StatusAccepted, common.Object{"async": true}
						})
						registerInstanceLastOPHandlers(globalBrokerServer, http.StatusInternalServerError, types.FAILED)
						rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

						waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)

						fullTree := fetchTree(rootID)
						validateAllResourceDeleted(ctx.SMRepository, fullTree.byResourceType)
						validateParentsRanAfterChildren(fullTree)
						validateDuplicationHasTheSameState(fullTree)
						validateAllOperationsHasTheSameState(fullTree, types.SUCCEEDED, tenantOperationsCount)
						validateNumberOfForceDeletions(fullTree, types.ServiceInstanceType, types.SUCCEEDED, 3)
					})
				})

				When("failed to remove instance using force", func() {
					It("should marked as failed", func() {
						registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
						rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

						ctx.SMRepository.Create(context.Background(), &types.ServiceBinding{
							Base: types.Base{
								ID:    "binding3",
								Ready: true,
							},
							ServiceInstanceID: osbInstanceID,
							Name:              "name",
						})

						waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), queryFailures)

						fullTree := fetchTree(rootID)
						validateParentsRanAfterChildren(fullTree)
						validateDuplicationHasTheSameState(fullTree)
						validateNumberOfForceDeletions(fullTree, types.ServiceBindingType, types.SUCCEEDED, 2)
						validateNumberOfForceDeletions(fullTree, types.ServiceInstanceType, types.FAILED, 1)

						Expect(strings.Contains(string(fullTree.root.Errors), "failed to force cascade delete of")).To(Equal(true))
					})
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

			When("getting deprovision errors", func() {
				When("succeed to delete instance using force", func() {
					It("should marked as succeeded", func() {
						globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
							return http.StatusBadRequest, common.Object{}
						})
						rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

						waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)

						fullTree := fetchTree(rootID)
						validateNumberOfForceDeletions(fullTree, types.ServiceInstanceType, types.SUCCEEDED, 1)
					})
				})

				When("failed to remove instance using force", func() {
					It("should marked as failed", func() {
						globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
							return http.StatusBadRequest, common.Object{}
						})
						rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)
						createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding3", map[string]interface{}{
							"service_id":        "global-service",
							"plan_id":           "global-plan",
							"organization_guid": "my-org",
						})

						waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), queryFailures)

						fullTree := fetchTree(rootID)
						validateNumberOfForceDeletions(fullTree, types.ServiceInstanceType, types.FAILED, 1)

						Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
							return operation.State == types.SUCCEEDED && !operationHasErrors(operation)
						})).To(Equal(2))
					})
				})
			})
		})
	})

	It("tenant with no children", func() {
		rootID := triggerCascadeOperation(context.Background(), types.TenantType, "some-tenant", true)
		waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)
	})

	Context("delete instance", func() {
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

		When("failed to remove instance using force", func() {
			It("should marked as failed", func() {
				globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
					return http.StatusBadRequest, common.Object{}
				})
				rootID := triggerCascadeOperation(context.Background(), types.ServiceInstanceType, osbInstanceID, true)
				createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding3", map[string]interface{}{
					"service_id":        "global-service",
					"plan_id":           "global-plan",
					"organization_guid": "my-org",
				})

				waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), queryFailures)
			})
		})

		When("getting deprovision errors", func() {
			It("should delete using force and marked as succeeded", func() {
				globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
					return http.StatusBadRequest, common.Object{}
				})
				rootID := triggerCascadeOperation(context.Background(), types.ServiceInstanceType, osbInstanceID, true)
				waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)
			})
		})
	})

	Context("delete binding", func() {
		BeforeEach(func() {
			createInstances = false
		})

		JustBeforeEach(func() {
			ctx.SMWithBasic.SetBasicCredentials(ctx, ctx.TestPlatform.Credentials.Basic.Username, ctx.TestPlatform.Credentials.Basic.Password)
			createOSBInstance(ctx, ctx.SMWithBasic, globalBrokerID, osbInstanceID, map[string]interface{}{
				"service_id":        "global-service",
				"plan_id":           "global-plan",
				"organization_guid": "my-org",
				"context": map[string]string{
					"tenant": tenantID,
				},
			})
			createOSBBinding(ctx, ctx.SMWithBasic, globalBrokerID, osbInstanceID, osbBindingID, map[string]interface{}{
				"service_id":        "global-service",
				"plan_id":           "global-plan",
				"organization_guid": "my-org",
			})
		})

		When("getting deprovision errors", func() {
			It("should delete using force and marked as succeeded", func() {
				globalBrokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
					return http.StatusBadRequest, common.Object{}
				})
				rootID := triggerCascadeOperation(context.Background(), types.ServiceBindingType, osbBindingID, true)
				waitCascadingProcessToFinish(actionTimeout+pollCascade, subaccountResourcesCount(), 1, queryForRoot(rootID), querySucceeded)
			})
		})
	})
})

func validateNumberOfForceDeletions(fullTree *tree, objectType types.ObjectType, state types.OperationState, countOfForcedOperations int) {
	By(fmt.Sprintf("validating there are %v %s that removed using force", countOfForcedOperations, objectType))
	Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
		return operation.ResourceType == objectType && operation.State == state && operationHasErrors(operation)
	})).To(Equal(countOfForcedOperations))
}

func operationHasErrors(operation *types.Operation) bool {
	return len(operation.Errors) > 2
}
