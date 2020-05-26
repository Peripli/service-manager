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
	"strconv"
	"time"
)

var _ = Describe("cascade operations", func() {

	BeforeEach(func() {
		cleanupInterval = 9999 * time.Hour
	})

	Context("deletion", func() {
		AfterEach(func() {
			ctx.Cleanup()
		})

		It("should succeed - cascade a big tenant tree", func() {
			subtreeCount := 3
			for i := 0; i < subtreeCount; i++ {
				registerSubaccountScopedPlatform(ctx, fmt.Sprintf("platform%s", strconv.Itoa(i*10)))
				broker2ID, _ := registerSubaccountScopedBroker(ctx, fmt.Sprintf("test-service%s", strconv.Itoa(i*10)), fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)))
				createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
					"name":            fmt.Sprintf("test-instance-smaap%s", strconv.Itoa(i*10)),
					"service_plan_id": plan.GetID(),
				})
				createOSBInstance(ctx, ctx.SMWithBasic, broker2ID, fmt.Sprintf("test-instance%s", strconv.Itoa(i*10)), map[string]interface{}{
					"service_id":        fmt.Sprintf("test-service%s", strconv.Itoa(i*10)),
					"plan_id":           fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)),
					"organization_guid": "my-org",
				})
				createOSBBinding(ctx, ctx.SMWithBasic, broker2ID, fmt.Sprintf("test-instance%s", strconv.Itoa(i*10)), fmt.Sprintf("binding%s", strconv.Itoa((i+1)*10)), map[string]interface{}{
					"service_id":        fmt.Sprintf("test-service%s", strconv.Itoa(i*10)),
					"plan_id":           fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)),
					"organization_guid": "my-org",
				})
				createOSBBinding(ctx, ctx.SMWithBasic, broker2ID, fmt.Sprintf("test-instance%s", strconv.Itoa(i*10)), fmt.Sprintf("binding%s", strconv.Itoa((i+1)*10+1)), map[string]interface{}{
					"service_id":        fmt.Sprintf("test-service%s", strconv.Itoa(i*10)),
					"plan_id":           fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)),
					"organization_guid": "my-org",
				})
			}

			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

			AssertOperationCount(func(count int) { Expect(count).To(Equal(3 + subtreeCount*3)) }, query.ByField(query.EqualsOperator, "parent_id", rootOpID))
			AssertOperationCount(func(count int) { Expect(count).To(Equal(tenantOperationsCount + subtreeCount*10)) }, queryForOperationsInTheSameTree)

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

			fullTree, err := fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})

		It("should succeed - cascade a platform", func() {
			triggerCascadeOperation(context.Background(), types.PlatformType, platformID, rootOpID)

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

			fullTree, err := fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})

		It("should fail - container failed to be deleted when cascade a platform", func() {
			registerInstanceLastOPHandlers(brokerServer, http.StatusInternalServerError, "")
			createContainerWithChildren()

			newCtx := context.WithValue(context.Background(), cascade.ContainerKey{}, "containerID")
			triggerCascadeOperation(newCtx, types.PlatformType, platformID, rootOpID)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)

			By("validating containerized errors collected")
			platformOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot)
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(platformOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(2))

			count := 0
			for _, e := range errors.Errors {
				if e.ParentType == types.ServiceInstanceType && e.ResourceType == types.ServiceInstanceType {
					count++
				}
			}
			Expect(count).To(Equal(1))
		})

		It("validate errors aggregated from bottom up", func() {
			registerBindingLastOPHandlers(brokerServer, http.StatusInternalServerError, types.FAILED)
			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot)
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

		It("should succeed - cascade a container", func() {
			containerID := createContainerWithChildren()

			newCtx := context.WithValue(context.Background(), cascade.ContainerKey{}, "containerID")
			triggerCascadeOperation(newCtx, types.ServiceInstanceType, containerID, rootOpID)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*3+pollCascade*3).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).NotTo(HaveOccurred())

			rootChildren := fullTree.byParentID[rootOpID]
			Expect(len(rootChildren)).To(Equal(1), "expected container has 1 instance")
			Expect(len(fullTree.byParentID[rootChildren[0].ID])).To(Equal(1), "expected instance has 1 binding")

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
			AssertOperationCount(func(count int) { Expect(count).To(Equal(3)) }, queryForOperationsInTheSameTree)
		})

		It("should fail - unsuccessful orphan mitigation", func() {
			pollingCount := 0
			brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
				if pollingCount == 0 {
					pollingCount++
					return http.StatusOK, common.Object{"state": "in progress"}
				} else {
					return http.StatusOK, common.Object{"state": "failed"}
				}
			})

			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

			By("validating binding failed and marked as orphan mitigation")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree,
					queryFailures,
					queryForOrphanMitigationOperations,
					queryForBindingsOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+pollCascade*2).Should(Equal(2))

			By("validating that instances without bindings were deleted")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree,
					querySucceeded,
					queryForInstanceOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+pollCascade*2).Should(Equal(2))

			By("validating bindings not in orphan mitigation")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree,
					queryForOrphanMitigationOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+maintainerRetry*2+cascadeOrphanMitigation*4).Should(Equal(0))

			By("validating root marked as failed")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*8+maintainerRetry*8).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})

		It("should succeed - successful orphan mitigation", func() {
			pollingCount := 0
			brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
				if pollingCount == 0 {
					pollingCount++
					return http.StatusOK, common.Object{"state": "in progress"}
				} else {
					return http.StatusOK, common.Object{"state": "failed"}
				}
			})

			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

			By("validating binding failed and marked as orphan mitigation")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree,
					queryFailures,
					queryForOrphanMitigationOperations,
					queryForBindingsOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+pollCascade*2).Should(Equal(2))

			By("validating that instances without bindings were deleted")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree,
					querySucceeded,
					queryForInstanceOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+pollCascade*2).Should(Equal(2))

			brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
				return http.StatusOK, common.Object{"state": "succeeded"}
			})

			By("validating bindings released from orphan mitigation and mark as succeeded")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForBindingsOperations,
					queryForOperationsInTheSameTree,
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+maintainerRetry*2+cascadeOrphanMitigation*4).Should(Equal(4))

			By("validating root is succeeded")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*8+maintainerRetry*8).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootOpID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})

		It("should failed - handle a stuck operation in cascade tree", func() {
			brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
				time.Sleep(100 * time.Millisecond)
				return http.StatusOK, common.Object{"state": "succeeded"}
			})

			triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

			instanceOPValue, err := ctx.SMRepository.Get(context.Background(), types.OperationType,
				query.ByField(query.EqualsOperator, "resource_id", "test-instance"),
				query.ByField(query.EqualsOperator, "cascade_root_id", rootOpID))

			Expect(err).NotTo(HaveOccurred())

			By("marking instance operation as stucked")
			instanceOP := instanceOPValue.(*types.Operation)
			instanceOP.Reschedule = false
			instanceOP.State = types.IN_PROGRESS
			_, err = ctx.SMRepository.Update(context.Background(), instanceOP, []*types.LabelChange{}, query.ByField(query.EqualsOperator, "id", instanceOP.ID))
			Expect(err).NotTo(HaveOccurred())

			By("validating operation activate orphan mitigation")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOrphanMitigationOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+maintainerRetry*2).Should(Equal(1))

			By("validating root marked as succeeded")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*10+pollCascade*10).Should(Equal(1))
		})

		Context("reconciliation", func() {
			BeforeEach(func() {
				reconciliation = 1 * time.Second
				actionTimeout = 500 * time.Millisecond
			})

			It("instance is reconciliation", func() {
				brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
					time.Sleep(1 * time.Second)
					return http.StatusOK, common.Object{"state": "in progress"}
				})

				triggerCascadeOperation(context.Background(), types.TenantType, tenantID, rootOpID)

				Eventually(func() int {
					count, err := ctx.SMRepository.Count(
						context.Background(),
						types.OperationType,
						queryForRoot,
						queryFailures)
					Expect(err).NotTo(HaveOccurred())

					return count
				}, actionTimeout*10+cleanupInterval*10+reconciliation*2).Should(Equal(1))
			})
		})
	})
})

func createContainerWithChildren() string {
	createOSBInstance(ctx, ctx.SMWithBasic, brokerID, "container-instance", map[string]interface{}{
		"service_id":        "test-service",
		"plan_id":           "plan-service",
		"organization_guid": "my-org",
	})
	containerInstanceID := createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
		"name":            "instance-in-container",
		"service_plan_id": plan.GetID(),
	})
	createSMAAPBinding(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
		"name":                "binding-in-container",
		"service_instance_id": containerInstanceID,
	})

	containerInstance, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "name", "container-instance"))
	Expect(err).NotTo(HaveOccurred())
	instanceInContainer, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", containerInstanceID))
	Expect(err).NotTo(HaveOccurred())

	change := types.LabelChange{
		Operation: "add",
		Key:       "containerID",
		Values:    []string{containerInstance.GetID()},
	}

	_, err = ctx.SMScheduler.ScheduleSyncStorageAction(context.TODO(), &types.Operation{
		Base: types.Base{
			ID:        "afasfasfasfasf",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Ready:     true,
		},
		Type:          types.UPDATE,
		State:         types.IN_PROGRESS,
		ResourceID:    instanceInContainer.GetID(),
		ResourceType:  types.ServiceInstanceType,
		CorrelationID: "-",
	}, func(ctx context.Context, repository storage.Repository) (object types.Object, e error) {
		return repository.Update(ctx, instanceInContainer, []*types.LabelChange{&change}, query.ByField(query.EqualsOperator, "id", instanceInContainer.GetID()))
	})
	Expect(err).NotTo(HaveOccurred())
	return "container-instance"
}

func validateDuplicationsWaited(fullTree *tree) {
	By("validating duplications waited and updated like sibling operators")
	for resourceID, operations := range fullTree.byResourceID {
		if resourceID == fullTree.root.ResourceID {
			continue
		}
		countOfOperationsThatProgressed := 0
		for _, operation := range operations {
			if operation.ExternalID != "" {
				countOfOperationsThatProgressed++
			}
		}
		Expect(countOfOperationsThatProgressed).
			To(Or(Equal(1), Equal(0)), fmt.Sprintf("resource: %s %s", operations[0].ResourceType, resourceID))
	}
}

func validateParentsRanAfterChildren(fullTree *tree) {
	By("validating parents ran after their children")
	for parentID, operations := range fullTree.byParentID {
		var parent *types.Operation
		if parentID == "" {
			parent = fullTree.root
		} else {
			parent = fullTree.byOperationID[parentID]
		}
		for _, operation := range operations {
			if !operation.DeletionScheduled.IsZero() {
				continue
			}
			Expect(parent.UpdatedAt.After(operation.UpdatedAt) || parent.UpdatedAt.Equal(operation.UpdatedAt)).
				To(BeTrue(), fmt.Sprintf("parent %s updateAt: %s is not after operation %s updateAt: %s", parent.ResourceType, parent.UpdatedAt, operation.ResourceType, operation.UpdatedAt))
		}
	}
}
