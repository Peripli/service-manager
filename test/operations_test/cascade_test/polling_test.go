package cascade_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
	"time"
)

var _ = Describe("Poll Cascade Delete", func() {

	Context("Cascade Delete", func() {

		AfterEach(func() {
			ctx.Cleanup()
		})

		It("big tenant tree should succeed", func() {
			subtreeCount := 5
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

		It("platform tree should succeed", func() {
			op := types.Operation{
				Base: types.Base{
					ID:        rootOpID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Description:   "bla",
				CascadeRootID: rootOpID,
				ResourceID:    platformID,
				Type:          types.DELETE,
				ResourceType:  types.PlatformType,
			}
			_, err := ctx.SMRepository.Create(context.TODO(), &op)
			Expect(err).NotTo(HaveOccurred())

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

		It("platform with container instance should fail", func() {
			registerInstanceLastOPHandlers(brokerServer, "failed")
			createOSBInstance(ctx, ctx.SMWithBasic, brokerID, "container-instance", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})
			createOSBInstance(ctx, ctx.SMWithBasic, brokerID, "child-instance", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})

			containerInstance, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "name", "container-instance"))
			Expect(err).NotTo(HaveOccurred())
			instanceInContainer, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "name", "child-instance"))
			Expect(err).NotTo(HaveOccurred())

			change := types.LabelChange{
				Operation: "add",
				Key:       "containerID",
				Values:    []string{containerInstance.GetID()},
			}
			_, err = ctx.SMRepository.Update(context.Background(), instanceInContainer, []*types.LabelChange{&change}, query.ByField(query.EqualsOperator, "id", instanceInContainer.GetID()))
			Expect(err).NotTo(HaveOccurred())

			op := types.Operation{
				Base: types.Base{
					ID:        rootOpID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Ready:     true,
				},
				Description:   "bla",
				CascadeRootID: rootOpID,
				ResourceID:    platformID,
				Type:          types.DELETE,
				ResourceType:  types.PlatformType,
			}
			newCtx := context.WithValue(context.Background(), cascade.ContainerKey{}, "containerID")
			_, err = ctx.SMRepository.Create(newCtx, &op)
			Expect(err).NotTo(HaveOccurred())

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					queryFailedOperations)
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
			registerBindingLastOPHandlers(brokerServer, types.FAILED)

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

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					queryFailedOperations)
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
				Expect(e.ParentID).To(Equal("test-instance"))
				Expect(e.ResourceID).To(Or(Equal("binding1"), Equal("binding2")))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Equal(types.ServiceBindingType))
				Expect(e.ParentType).To(Equal(types.ServiceInstanceType))
			}
		})

		It("Cascade container", func() {

		})

		It("Activate a orphan mitigation for instance and expect for failures", func() {

		})

		It("Handle a stuck operation in cascade tree", func() {

		})
	})
})

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
