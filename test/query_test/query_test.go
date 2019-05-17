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

	Context("with 2 notification created at different times", func() {
		var now time.Time
		var id1, id2 string
		BeforeEach(func() {
			now = time.Now()

			id1 = createNotification(repository, now)
			id2 = createNotification(repository, now.Add(-30*time.Minute))
		})

		It("notifications older than the provided time should not be found", func() {
			operand := now.Add(-time.Hour).Format(time.RFC3339)
			criteria := query.ByField(query.LessThanOperator, "created_at", operand)
			list := queryNotification(repository, criteria)
			expectNotifications(list)
		})

		It("only 1 should be older than the provided time", func() {
			operand := now.Add(-20 * time.Minute).Format(time.RFC3339)
			criteria := query.ByField(query.LessThanOperator, "created_at", operand)
			list := queryNotification(repository, criteria)
			expectNotifications(list, id2)
		})

		It("2 notifications should be found newer than the provided time", func() {
			operand := now.Add(-time.Hour).Format(time.RFC3339)
			criteria := query.ByField(query.GreaterThanOperator, "created_at", operand)
			list := queryNotification(repository, criteria)
			expectNotifications(list, id1, id2)
		})

		It("only 1 notifications should be found newer than the provided time", func() {
			operand := now.Add(-10 * time.Minute).Format(time.RFC3339)
			criteria := query.ByField(query.GreaterThanOperator, "created_at", operand)
			list := queryNotification(repository, criteria)
			expectNotifications(list, id1)
		})

		It("no notifications should be found newer than the last one created", func() {
			operand := now.Add(10 * time.Minute).Format(time.RFC3339)
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
