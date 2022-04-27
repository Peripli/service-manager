package query_test

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestQuery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Query Tests Suite")
}

var _ = Describe("Service Manager Query", func() {
	var ctx *common.TestContext
	var repository storage.Repository

	BeforeSuite(func() {
		ctx = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			repository = smb.Storage
			return nil
		}).Build()
	})

	AfterEach(func() {
		if repository != nil {
			err := repository.Delete(context.Background(), types.NotificationType)
			Expect(err).ShouldNot(HaveOccurred())
		}
	})

	AfterSuite(func() {
		if ctx != nil {
			ctx.Cleanup()
		}
	})

	Context("Named Query", func() {
		Context("Service instance and last operations query test", func() {
			var serviceInstance1, serviceInstance2 *types.ServiceInstance
			BeforeEach(func() {
				_, serviceInstance1 = common.CreateInstanceInPlatform(ctx, ctx.TestPlatform.ID)
				_, serviceInstance2 = common.CreateInstanceInPlatform(ctx, ctx.TestPlatform.ID)
			})

			AfterEach(func() {
				ctx.CleanupAdditionalResources()
			})

			It("should return the last operation for a newly created resource", func() {
				queryParams := map[string]interface{}{
					"id_list":       []string{serviceInstance1.ID},
					"resource_type": types.ServiceInstanceType,
				}

				list, err := repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
				Expect(err).ShouldNot(HaveOccurred())
				lastOperation := list.ItemAt(0).(*types.Operation)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(list.Len()).To(BeEquivalentTo(1))
				Expect(lastOperation.State).To(Equal(types.SUCCEEDED))
				Expect(lastOperation.Type).To(Equal(types.CREATE))
				Expect(lastOperation.ResourceID).To(Equal(serviceInstance1.ID))
			})

			When("new operation is created for the resource", func() {
				BeforeEach(func() {
					operation := &types.Operation{
						Base: types.Base{
							ID:        "my_test_op_latest",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							Labels:    make(map[string][]string),
							Ready:     true,
						},
						Description:       "my_test_op_latest",
						Type:              types.CREATE,
						State:             types.SUCCEEDED,
						ResourceID:        serviceInstance1.ID,
						ResourceType:      web.ServiceInstancesURL,
						CorrelationID:     "test-correlation-id",
						Reschedule:        false,
						DeletionScheduled: time.Time{},
					}

					_, err := repository.Create(context.Background(), operation)
					Expect(err).ShouldNot(HaveOccurred())
				})
				It("should return the new operation as last operation", func() {
					queryParams := map[string]interface{}{
						"id_list":       []string{serviceInstance1.ID},
						"resource_type": types.ServiceInstanceType,
					}
					list, err := repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
					Expect(err).ShouldNot(HaveOccurred())
					lastOperation := list.ItemAt(0).(*types.Operation)
					Expect(list.Len()).To(BeEquivalentTo(1))
					Expect(lastOperation.State).To(Equal(types.SUCCEEDED))
					Expect(list.Len()).To(BeEquivalentTo(1))
					Expect(lastOperation.ID).To(Equal("my_test_op_latest"))
				})

			})

			When("Operations exists for other resources", func() {

				BeforeEach(func() {
					operation := &types.Operation{
						Base: types.Base{
							ID:        "my_test_op_latest_instance2",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							Labels:    make(map[string][]string),
							Ready:     true,
						},
						Description:       "my_test_op_latest",
						Type:              types.UPDATE,
						State:             types.SUCCEEDED,
						ResourceID:        serviceInstance2.ID,
						ResourceType:      web.ServiceInstancesURL,
						CorrelationID:     "test-correlation-id",
						Reschedule:        false,
						DeletionScheduled: time.Time{},
					}

					_, err := repository.Create(context.Background(), operation)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should return only the last operations associated to the resource in query", func() {
					queryParams := map[string]interface{}{
						"id_list":       []string{serviceInstance1.ID},
						"resource_type": types.ServiceInstanceType,
					}
					list, err := repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(list.Len()).To(BeEquivalentTo(1))
					Expect(list.ItemAt(0).GetID()).ToNot(Equal("my_test_op_latest_instance2"))
				})

			})

			It("should return the last operation for every instances in the query", func() {
				queryParams := map[string]interface{}{
					"id_list":       []string{serviceInstance2.ID, serviceInstance1.ID},
					"resource_type": types.ServiceInstanceType,
				}
				list, err := repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(err).ShouldNot(HaveOccurred())
				Expect(list.Len()).To(BeEquivalentTo(2))
				lastOperation1 := list.ItemAt(0).(*types.Operation)
				lastOperation2 := list.ItemAt(1).(*types.Operation)
				Expect(lastOperation1.State).To(Equal(types.SUCCEEDED))
				Expect(lastOperation2.State).To(Equal(types.SUCCEEDED))
			})
		})

	})

	Context("ByExists/ByNotExists SubQuery", func() {
		Context("ByExists", func() {
			When("there are multiple operations for each instance", func() {
				BeforeEach(func() {
					err := common.RemoveAllInstances(ctx)
					Expect(err).ShouldNot(HaveOccurred())
					common.RemoveAllOperations(ctx.SMRepository)

					serviceInstance1 := &types.ServiceInstance{
						Base: types.Base{
							ID: "instance1",
						},
					}
					serviceInstance2 := &types.ServiceInstance{
						Base: types.Base{
							ID: "instance2",
						},
					}
					oldestOpForInstance1 := &types.Operation{
						Base: types.Base{
							ID: "oldestOpForInstance1",
						},
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   serviceInstance1.ID,
						ResourceType: web.ServiceInstancesURL,
					}
					latestOpForInstance1 := &types.Operation{
						Base: types.Base{
							ID: "latestOpForInstance1",
						},
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   serviceInstance1.ID,
						ResourceType: web.ServiceInstancesURL,
					}
					oldestOpForInstance2 := &types.Operation{
						Base: types.Base{
							ID: "oldestOpForInstance2",
						},
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   serviceInstance2.ID,
						ResourceType: web.ServiceInstancesURL,
					}
					latestOpForInstance2 := &types.Operation{
						Base: types.Base{
							ID: "latestOpForInstance2",
						},
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   serviceInstance2.ID,
						ResourceType: web.ServiceInstancesURL,
					}

					_, err = repository.Create(context.Background(), serviceInstance1)
					Expect(err).ShouldNot(HaveOccurred())
					_, err = repository.Create(context.Background(), serviceInstance2)
					Expect(err).ShouldNot(HaveOccurred())
					_, err = repository.Create(context.Background(), oldestOpForInstance1)
					Expect(err).ShouldNot(HaveOccurred())
					_, err = repository.Create(context.Background(), latestOpForInstance1)
					Expect(err).ShouldNot(HaveOccurred())
					_, err = repository.Create(context.Background(), oldestOpForInstance2)
					Expect(err).ShouldNot(HaveOccurred())
					_, err = repository.Create(context.Background(), latestOpForInstance2)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should return only the operations being last operation for their corresponding instances", func() {
					criteria := []query.Criterion{query.ByExists(storage.GetSubQuery(storage.QueryForAllLastOperationsPerResource))}

					list, err := repository.List(context.Background(), types.OperationType, criteria...)
					Expect(err).ToNot(HaveOccurred())
					Expect(list.Len()).To(BeEquivalentTo(2))
					Expect(listContains(list, "latestOpForInstance1"))
					Expect(listContains(list, "latestOpForInstance2"))
				})

				It("should return only the operations being NOT last operation for their corresponding instances", func() {
					criteria := []query.Criterion{query.BySubquery(query.InSubqueryOperator, "id", storage.GetSubQuery(storage.QueryForAllNotLastOperationsPerResource))}

					list, err := repository.List(context.Background(), types.OperationType, criteria...)
					Expect(err).ToNot(HaveOccurred())
					Expect(list.Len()).To(BeEquivalentTo(2))
					Expect(listContains(list, "oldestOpForInstance1"))
					Expect(listContains(list, "oldestOpForInstance2"))
				})
			})

		})
		Context("ByExists with template parameters", func() {
			When("There is an operation is associated to a resource and another operation that is resource-less", func() {
				BeforeEach(func() {
					err := common.RemoveAllInstances(ctx)
					Expect(err).ShouldNot(HaveOccurred())
					common.RemoveAllOperations(ctx.SMRepository)

					resource := &types.Platform{
						Base: types.Base{ID: "test-resource"},
					}
					opForInstance1 := &types.Operation{
						Base: types.Base{
							ID: "opForInstance1",
						},
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   "test-resource",
						ResourceType: web.ServiceInstancesURL,
					}
					resourcelessOperation := &types.Operation{
						Base: types.Base{
							ID: "resourcelessOp",
						},
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   "NON_EXISTENT_RESOURCE",
						ResourceType: web.ServiceInstancesURL,
					}

					_, err = repository.Create(context.Background(), resource)
					Expect(err).ShouldNot(HaveOccurred())
					_, err = repository.Create(context.Background(), opForInstance1)
					Expect(err).ToNot(HaveOccurred())
					_, err = repository.Create(context.Background(), resourcelessOperation)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should retrieve only the operation that is not associated to a resource", func() {
					params := storage.SubQueryParams{
						"RESOURCE_TABLE": "platforms",
					}
					subQuery, err := storage.GetSubQueryWithParams(storage.QueryForOperationsWithResource, params)
					Expect(err).ToNot(HaveOccurred())
					criterion := query.ByNotExists(subQuery)
					criteria := []query.Criterion{criterion}

					list, err := repository.List(context.Background(), types.OperationType, criteria...)
					Expect(err).ToNot(HaveOccurred())
					Expect(list.Len()).To(BeEquivalentTo(1))
					Expect(listContains(list, "resourcelessOp"))
				})

				It("should retrieve only the operation that is associated to a resource", func() {
					params := storage.SubQueryParams{
						"RESOURCE_TABLE": "platforms",
					}

					subQuery, err := storage.GetSubQueryWithParams(storage.QueryForOperationsWithResource, params)
					Expect(err).ToNot(HaveOccurred())
					criterion := query.ByExists(subQuery)
					criteria := []query.Criterion{criterion}

					list, err := repository.List(context.Background(), types.OperationType, criteria...)
					Expect(err).ToNot(HaveOccurred())
					Expect(list.Len()).To(BeEquivalentTo(1))
					Expect(listContains(list, "opForInstance1"))
				})
			})

			When("There are tenant specific services", func() {
				var tenantServiceMap map[string]interface{}
				BeforeEach(func() {
					tenantPlan := common.GenerateFreeTestPlan()
					globalPlan := common.GenerateFreeTestPlan()
					tenantService := common.GenerateTestServiceWithPlans(tenantPlan)
					globalService := common.GenerateTestServiceWithPlans(globalPlan)
					tenantServiceMap = make(map[string]interface{})
					if err := json.Unmarshal([]byte(tenantService), &tenantServiceMap); err != nil {
						panic(err)
					}

					tenantCatalog := common.NewEmptySBCatalog()
					tenantCatalog.AddService(tenantService)
					labels := common.Object{
						"labels": common.Object{
							"tenant_id": common.Array{"tenant_id_value"},
						},
					}
					ctx.RegisterBrokerWithCatalogAndLabels(tenantCatalog, labels, http.StatusCreated)
					globalCatalog := common.NewEmptySBCatalog()
					globalCatalog.AddService(globalService)
					ctx.RegisterBrokerWithCatalog(globalCatalog)
				})

				It("should find query for tenant scoped service offerings", func() {
					params := storage.SubQueryParams{
						"TENANT_KEY": "tenant_id",
					}
					subQuery, err := storage.GetSubQueryWithParams(storage.QueryForTenantScopedServiceOfferings, params)
					Expect(err).ToNot(HaveOccurred())
					criteria := []query.Criterion{query.ByExists(subQuery)}

					list, err := repository.List(context.Background(), types.ServiceOfferingType, criteria...)
					Expect(err).ToNot(HaveOccurred())
					Expect(list.Len()).To(BeEquivalentTo(1))
					service := list.ItemAt(0).(*types.ServiceOffering)
					Expect(service.CatalogID).To(Equal(tenantServiceMap["id"]))
				})
			})
		})
	})

	Context("Query Notifications", func() {
		Context("with 2 notification created at different times", func() {
			var now time.Time
			var id1, id2 string
			BeforeEach(func() {
				now = time.Now()

				id1 = createNotification(repository, now)
				id2 = createNotification(repository, now.Add(-30*time.Minute))
			})

			It("notifications older than the provided time should not be found", func() {
				operand := util.ToRFCNanoFormat(now.Add(-time.Hour))
				criteria := query.ByField(query.LessThanOperator, "created_at", operand)
				list := queryNotification(repository, criteria)
				expectNotifications(list)
			})

			It("only 1 should be older than the provided time", func() {
				operand := util.ToRFCNanoFormat(now.Add(-20 * time.Minute))
				criteria := query.ByField(query.LessThanOperator, "created_at", operand)
				list := queryNotification(repository, criteria)
				expectNotifications(list, id2)
			})

			It("2 notifications should be found newer than the provided time", func() {
				operand := util.ToRFCNanoFormat(now.Add(-time.Hour))
				criteria := query.ByField(query.GreaterThanOperator, "created_at", operand)
				list := queryNotification(repository, criteria)
				expectNotifications(list, id1, id2)
			})

			It("only 1 notifications should be found newer than the provided time", func() {
				operand := util.ToRFCNanoFormat(now.Add(-10 * time.Minute))
				criteria := query.ByField(query.GreaterThanOperator, "created_at", operand)
				list := queryNotification(repository, criteria)
				expectNotifications(list, id1)
			})

			It("no notifications should be found newer than the last one created", func() {
				operand := util.ToRFCNanoFormat(now.Add(10 * time.Minute))
				criteria := query.ByField(query.GreaterThanOperator, "created_at", operand)
				list := queryNotification(repository, criteria)
				expectNotifications(list)
			})
		})
	})

})

func expectNotifications(list types.ObjectList, ids ...string) {
	Expect(list.Len()).To(Equal(len(ids)))
	for i := 0; i < list.Len(); i++ {
		Expect(ids).To(ContainElement(list.ItemAt(i).GetID()))
	}
}

func createNotification(repository storage.Repository, createdAt time.Time) string {
	uid, err := uuid.NewV4()
	Expect(err).ShouldNot(HaveOccurred())

	notification := &types.Notification{
		Base: types.Base{
			ID:        uid.String(),
			CreatedAt: createdAt,
			Ready:     true,
		},
		Payload:  []byte("{}"),
		Resource: "empty",
		Type:     "CREATED",
	}
	_, err = repository.Create(context.Background(), notification)
	Expect(err).ShouldNot(HaveOccurred())

	return notification.ID
}

func listContains(list types.ObjectList, itemId string) bool {
	for i := 0; i < list.Len(); i++ {
		if list.ItemAt(i).GetID() == itemId {
			return true
		}
	}
	return false
}

func queryNotification(repository storage.Repository, criterias ...query.Criterion) types.ObjectList {
	list, err := repository.List(context.Background(), types.NotificationType, criterias...)
	Expect(err).ShouldNot(HaveOccurred())
	return list
}
