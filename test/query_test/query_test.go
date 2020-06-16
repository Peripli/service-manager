package query_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/web"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/util"

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

	FContext ("Service instance and last operations query test",func(){
		var serviceInstance1,serviceInstance2 *types.ServiceInstance
		BeforeEach(func(){
			_, serviceInstance1 = common.CreateInstanceInPlatform(ctx, ctx.TestPlatform.ID)
			_, serviceInstance2 = common.CreateInstanceInPlatform(ctx, ctx.TestPlatform.ID)
		})

		AfterEach(func(){
			ctx.CleanupAdditionalResources()
		})

		It("Last operation is returned for a newly created instance", func() {
			queryParams := map[string]interface{}{
				"id_list": []string{serviceInstance1.ID},
			}

			list, err := repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
			Expect(err).ShouldNot(HaveOccurred())
			lastOperation := list.ItemAt(0).(*types.Operation)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(list.Len()).To(BeEquivalentTo(1))
			Expect(lastOperation.State).To(Equal(types.SUCCEEDED))
		})

		It("After a new operation is added to an instance", func() {
			queryParams := map[string]interface{}{
				"id_list": []string{serviceInstance1.ID},
			}

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
				State:             types.IN_PROGRESS,
				ResourceID:        serviceInstance1.ID,
				ResourceType:      web.ServiceInstancesURL,
				CorrelationID:     "test-correlation-id",
				Reschedule:        false,
				DeletionScheduled: time.Time{},
			}

			_, err := repository.Create(context.Background(), operation)
			Expect(err).ShouldNot(HaveOccurred())

			list, err := repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
			Expect(err).ShouldNot(HaveOccurred())
			lastOperation := list.ItemAt(0).(*types.Operation)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(list.Len()).To(BeEquivalentTo(1))
			Expect(lastOperation.State).To(Equal(types.IN_PROGRESS))
			Expect(list.Len()).To(BeEquivalentTo(1))
			Expect(lastOperation.ID).To(Equal("my_test_op_latest"))
		})


		It("The last operation for every instances in query is returned", func() {
			queryParams := map[string]interface{}{
				"id_list": []string{serviceInstance2.ID,serviceInstance1.ID},
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

		It("Correct last operation is returned after old operations are removed", func() {
			queryParams := map[string]interface{}{
				"id_list": []string{serviceInstance1.ID},
			}


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
				State:             types.IN_PROGRESS,
				ResourceID:        serviceInstance1.ID,
				ResourceType:      web.ServiceInstancesURL,
				CorrelationID:     "test-correlation-id",
				Reschedule:        false,
				DeletionScheduled: time.Time{},
			}

			_, err := repository.Create(context.Background(), operation)

			list, err := repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(list.Len()).To(BeEquivalentTo(1))
			lastOperation := list.ItemAt(0).(*types.Operation)
			Expect(lastOperation.ID).To(Equal("my_test_op_latest"))

			criteria := query.ByField(query.EqualsOperator, "id", "my_test_op_latest")
			err = repository.Delete(context.Background(), types.OperationType,criteria)
			Expect(err).ShouldNot(HaveOccurred())

			list, err = repository.QueryForList(context.Background(), types.OperationType, storage.QueryForLastOperationsPerResource, queryParams)
			lastOperation = list.ItemAt(0).(*types.Operation)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(list.Len()).To(BeEquivalentTo(1))
			Expect(lastOperation.ID).To(Not(Equal("my_test_op_latest")));
		})
	})
	Context("with 2 notification created at different times", func() {
		var now time.Time
		var id1, id2 string
		BeforeEach(func() {
			now = time.Now()

			id1 = createNotification(repository, now)
			id2 = createNotification(repository, now.Add(-30*time.Minute))
		})

		It("finds objects by ID in", func() {
			args := map[string]interface{}{
				"id_list": []string{id1, id2},
				"type":    "CREATED",
			}
			list, err := repository.QueryForList(context.Background(), types.NotificationType, storage.QueryByTypeAndIDIn, args)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(list.Len()).To(BeEquivalentTo(2))
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

func queryNotification(repository storage.Repository, criterias ...query.Criterion) types.ObjectList {
	list, err := repository.List(context.Background(), types.NotificationType, criterias...)
	Expect(err).ShouldNot(HaveOccurred())
	return list
}
