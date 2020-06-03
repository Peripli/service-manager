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
	Context("tenant tree", func() {

		JustBeforeEach(func() {
			initTenantResources(true)
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

			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID)

			AssertOperationCount(func(count int) { Expect(count).To(Equal(3 + subtreeCount*3)) }, query.ByField(query.EqualsOperator, "parent_id", rootID))
			AssertOperationCount(func(count int) { Expect(count).To(Equal(tenantOperationsCount + subtreeCount*10)) }, queryForOperationsInTheSameTree(rootID))

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
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

			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID)

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
			}, actionTimeout*2+pollCascade*2).Should(Equal(2))

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
			}, actionTimeout*2+pollCascade*2).Should(Equal(2))

			By("validating bindings not in orphan mitigation")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForOperationsInTheSameTree(rootID),
					queryForOrphanMitigationOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*2+maintainerRetry*2+cascadeOrphanMitigation*4).Should(Equal(0))

			By("validating root marked as failed")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*8+maintainerRetry*8).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
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

			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID)

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
			validateDuplicationsWaited(fullTree)
		})

		It("should failed - handle a stuck operation in cascade tree", func() {
			brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
				time.Sleep(100 * time.Millisecond)
				return http.StatusOK, common.Object{"state": "succeeded"}
			})

			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID)

			instanceOPValue, err := ctx.SMRepository.Get(context.Background(), types.OperationType,
				query.ByField(query.EqualsOperator, "resource_id", "test-instance"),
				query.ByField(query.EqualsOperator, "cascade_root_id", rootID))

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
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*10+pollCascade*10).Should(Equal(1))
		})
	})

	Context("platform tree", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("should succeed - cascade a platform", func() {
			rootID := triggerCascadeOperation(context.Background(), types.PlatformType, platformID)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})
	})

	Context("broker tree", func() {
		JustBeforeEach(func() {
			initTenantResources(false)
		})

		It("should succeeded - cascade broker without children", func() {
			rootID := triggerCascadeOperation(context.Background(), types.ServiceBrokerType, brokerID)

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
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})
	})

	Context("errors", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("validate errors aggregated from bottom up", func() {
			registerBindingLastOPHandlers(brokerServer, http.StatusInternalServerError, types.FAILED)
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).NotTo(HaveOccurred())

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
	})

	Context("container", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("should fail - container failed to be deleted when cascade a platform", func() {
			registerInstanceLastOPHandlers(brokerServer, http.StatusInternalServerError, "")
			createContainerWithChildren()

			newCtx := context.WithValue(context.Background(), cascade.ParentInstanceLabelKey{}, "containerID")
			rootID := triggerCascadeOperation(newCtx, types.PlatformType, platformID)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)

			By("validating containerized errors collected")
			platformOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
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

		It("should succeed - cascade a container", func() {
			containerID := createContainerWithChildren()

			newCtx := context.WithValue(context.Background(), cascade.ParentInstanceLabelKey{}, "containerID")
			rootID := triggerCascadeOperation(newCtx, types.ServiceInstanceType, containerID)

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
			Expect(err).NotTo(HaveOccurred())

			rootChildren := fullTree.byParentID[rootID]
			Expect(len(rootChildren)).To(Equal(1), "expected container has 1 instance")
			Expect(len(fullTree.byParentID[rootChildren[0].ID])).To(Equal(1), "expected instance has 1 binding")

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
			AssertOperationCount(func(count int) { Expect(count).To(Equal(3)) }, queryForOperationsInTheSameTree(rootID))
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
	By("validating duplications waited and updated like sibling operations")
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
