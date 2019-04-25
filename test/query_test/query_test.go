package query_test

import (
	"context"
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
			_, err := repository.Delete(context.Background(), types.NotificationType)
			Expect(err).ShouldNot(HaveOccurred())
		}
	})

	AfterSuite(func() {
		if ctx != nil {
			ctx.Cleanup()
		}
	})

	Context("when search with greater than operator", func() {
		It("notifications with less than value should not be found", func() {
			now := time.Now()
			id1 := createNotification(repository, now)
			id2 := createNotification(repository, now.Add(-30*time.Minute))

			operand := now.Add(-time.Hour).Format(time.RFC3339)
			criteria := query.ByField(query.LessThanOperator, "created_at", operand)
			list := queryNotification(repository, criteria)
			expectNotifications(list)

			operand = now.Add(-20 * time.Minute).Format(time.RFC3339)
			criteria = query.ByField(query.LessThanOperator, "created_at", operand)
			list = queryNotification(repository, criteria)
			expectNotifications(list, id2)

			operand = now.Add(-time.Hour).Format(time.RFC3339)
			criteria = query.ByField(query.GreaterThanOperator, "created_at", operand)
			list = queryNotification(repository, criteria)
			expectNotifications(list, id1, id2)

			operand = now.Add(-2 * time.Hour).Format(time.RFC3339)
			criteria = query.ByField(query.GreaterThanOperator, "created_at", operand)
			list = queryNotification(repository, criteria)
			expectNotifications(list, id1, id2)

			operand = now.Add(-10 * time.Minute).Format(time.RFC3339)
			criteria = query.ByField(query.GreaterThanOperator, "created_at", operand)
			list = queryNotification(repository, criteria)
			expectNotifications(list, id1)

			operand = now.Add(10 * time.Minute).Format(time.RFC3339)
			criteria = query.ByField(query.GreaterThanOperator, "created_at", operand)
			list = queryNotification(repository, criteria)
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
