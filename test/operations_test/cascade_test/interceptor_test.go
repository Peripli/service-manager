package cascade_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

type childrenMap = map[string]interface{}

var _ = Describe("Cascade Operation Interceptor", func() {

	JustBeforeEach(func() {
		initTenantResources(true)
	})

	Context("tree creation", func() {

		Context("should fail", func() {
			It("not valid op root not equals op id", func() {
				op := types.Operation{
					Base: types.Base{
						ID:        generateID(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:   "bla",
					ResourceID:    tenantID,
					CascadeRootID: "fake-id",
					Type:          types.DELETE,
					ResourceType:  types.TenantType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).To(HaveOccurred())
				expectedErrMsg := fmt.Sprintf("root operation should have the same CascadeRootID and ID")
				Expect(err.Error()).To(Equal(expectedErrMsg))
			})

			It("resourceID not exists", func() {
				rootID := generateID()
				op := types.Operation{
					Base: types.Base{
						ID:        rootID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:   "bla",
					ResourceID:    "fake-resource",
					CascadeRootID: rootID,
					Type:          types.DELETE,
					ResourceType:  types.PlatformType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).To(HaveOccurred())
				expectedErrMsg := fmt.Sprintf("not found")
				Expect(err.Error()).To(Equal(expectedErrMsg))
			})

		})

		Context("should skip", func() {
			It("not cascade ops", func() {
				rootID := generateID()
				op := types.Operation{
					Base: types.Base{
						ID:        rootID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:  "bla",
					ResourceID:   tenantID,
					State:        types.PENDING,
					Type:         types.DELETE,
					ResourceType: types.TenantType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				AssertOperationCount(func(count int) { Expect(count).To(Equal(1)) }, query.ByField(query.EqualsOperator, "resource_id", tenantID))
				AssertOperationCount(func(count int) { Expect(count).To(Equal(0)) }, query.ByField(query.EqualsOperator, "parent_id", rootID))
			})

			It("op type different than delete", func() {
				rootID := generateID()
				op := types.Operation{
					Base: types.Base{
						ID:        rootID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:  "bla",
					ResourceID:   tenantID,
					State:        types.PENDING,
					Type:         types.CREATE,
					ResourceType: types.TenantType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				AssertOperationCount(func(count int) { Expect(count).To(Equal(1)) }, query.ByField(query.EqualsOperator, "resource_id", tenantID))
				AssertOperationCount(func(count int) { Expect(count).To(Equal(0)) }, query.ByField(query.EqualsOperator, "parent_id", rootID))
			})

			It("operation for the same resource exists", func() {
				rootID := generateID()
				op := types.Operation{
					Base: types.Base{
						ID:        rootID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Description:   "bla",
					ResourceID:    tenantID,
					CascadeRootID: rootID,
					Type:          types.DELETE,
					ResourceType:  types.TenantType,
				}

				_, err := ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				op.ID = "some-id"
				op.CascadeRootID = "some-id"
				_, err = ctx.SMRepository.Create(context.TODO(), &op)
				Expect(err).NotTo(HaveOccurred())
				AssertOperationCount(func(count int) { Expect(count).To(Equal(1)) }, query.ByField(query.EqualsOperator, "resource_id", tenantID))
			})
		})

		Context("should succeed", func() {
			It("validate tree hierarchy", func() {
				container := createContainerWithChildren()
				newCtx := context.WithValue(context.Background(), cascade.ParentInstanceLabelKeys{}, []string{"containerID"})
				rootID := triggerCascadeOperation(newCtx, types.TenantType, tenantID, false)

				tree, err := fetchFullTree(ctx.SMRepository, rootID)
				Expect(err).ToNot(HaveOccurred())

				containerInstance := container.instances[0]
				containerBinding := container.bindingForInstance[containerInstance][0]
				expectedTree := childrenMap{
					tenantID: childrenMap{
						platformID: childrenMap{
							"tenant_platform_global_broker": true,
							osbInstanceID: childrenMap{
								"binding1": true,
								"binding2": true,
							},
							container.id: childrenMap{
								containerInstance: childrenMap{
									containerBinding: true,
								},
							},
						},
						tenantBrokerID: childrenMap{
							osbInstanceID: childrenMap{
								"binding1": true,
								"binding2": true,
							},
							containerInstance: childrenMap{
								containerBinding: true,
							},
							"global_platform_tenant_broker": true,
							smaapInstanceID1:                true,
						},
						"global_platform_global_broker": true,
						smaapInstanceID2:                true,
					},
				}

				validateTreeHierarchy(tree, rootID, expectedTree)
			})

			It("empty virtual operation", func() {
				triggerCascadeOperation(context.Background(), types.TenantType, "fake-tenant-id", false)
				AssertOperationCount(func(count int) { Expect(count).To(Equal(2)) }, query.ByField(query.EqualsOperator, "resource_id", "fake-tenant-id"))
			})

			It("virtual cascade op", func() {
				rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, false)

				tree, err := fetchFullTree(ctx.SMRepository, rootID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tree.byOperationID)).To(Equal(tenantOperationsCount))

				platformOpID := tree.byResourceID[platformID][0].ID
				brokerOpID := tree.byResourceID[tenantBrokerID][0].ID
				instanceOpID := tree.byResourceID[osbInstanceID][0].ID

				// Tenant[broker, platform , smaap_instance]
				Expect(len(tree.byParentID[rootID])).To(Equal(4))
				// Platform [instance]
				Expect(len(tree.byParentID[platformOpID])).To(Equal(2))
				// Broker[instance, smaap_instance]
				Expect(len(tree.byParentID[brokerOpID])).To(Equal(3))
				// Instance[binding1, binding2]
				Expect(len(tree.byParentID[instanceOpID])).To(Equal(2))

			})

			It("non virtual cascade op", func() {
				rootID := triggerCascadeOperation(context.Background(), types.ServiceBrokerType, tenantBrokerID, false)
				tree, err := fetchFullTree(ctx.SMRepository, rootID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(tree.byOperationID)).To(Equal(6))
				// Broker[instance, smaap_instance]
				Expect(len(tree.byParentID[rootID])).To(Equal(3))

				instanceOpID := tree.byResourceID[osbInstanceID][0].ID
				// Instance[binding1, binding2]
				Expect(len(tree.byParentID[instanceOpID])).To(Equal(2))
			})
		})

	})
})

func validateTreeHierarchy(fullTree *tree, parentID string, expectedTree childrenMap) {
	resourceID := fullTree.byOperationID[parentID].ResourceID
	_, found := expectedTree[resourceID]
	Expect(found).To(Equal(true), fmt.Sprintf("resource %s does not exist", resourceID))

	for _, operation := range fullTree.byParentID[parentID] {
		if _, isBool := expectedTree[operation.ID].(bool); !isBool {
			validateTreeHierarchy(fullTree, operation.ID, expectedTree[resourceID].(childrenMap))
		}
	}

	if _, isBool := expectedTree[resourceID].(bool); !isBool {
		Expect(len(expectedTree[resourceID].(childrenMap))).To(Equal(0), fmt.Sprintf("%v not found", expectedTree[resourceID].(childrenMap)))
	}
	delete(expectedTree, resourceID)
}
