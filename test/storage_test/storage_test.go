package storage_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
)

func TestStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Suite")
}

var _ = Describe("Test", func() {
	var ctx *common.TestContext
	var platform types.Object
	var err error

	BeforeEach(func() {
		ctx = common.NewTestContextBuilder().Build()
		platform, err = ctx.SMRepository.Create(context.Background(), &types.Platform{
			Base: types.Base{
				ID: "id",
			},
			Description: "desc",
			Name:        "platform_name",
			Type:        "cloudfoundry",
		})
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	Context("when resource is created without labels", func() {
		It("should be fetched from storage without any labels", func() {
			byID := query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err := ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(obj.GetLabels()).To(HaveLen(0))
		})
	})

	Context("when platform resource is updated in transaction", func() {
		It("does not delete its visibilities", func() {
			ctx.RegisterBroker()
			plans := ctx.SMWithBasic.List(web.ServicePlansURL)
			planID := plans.First().Object().Value("id").String().Raw()
			visibility := types.Visibility{
				PlatformID:    platform.GetID(),
				ServicePlanID: planID,
			}

			visibilityID := ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(visibility).Expect().
				Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

			err := ctx.SMRepository.InTransaction(context.TODO(), func(ctx context.Context, storage storage.Repository) error {
				var updatedPlatform types.Object
				byID := query.ByField(query.EqualsOperator, "id", platform.GetID())
				platformFromStorage, err := storage.Get(ctx, types.PlatformType, byID)
				Expect(err).ToNot(HaveOccurred())

				platformFromStorage.(*types.Platform).Active = true
				if updatedPlatform, err = storage.Update(ctx, platformFromStorage, query.LabelChanges{}); err != nil {
					return err
				}
				Expect(updatedPlatform.(*types.Platform).Active).To(Equal(true))
				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			ctx.SMWithOAuth.GET(web.VisibilitiesURL + "/" + visibilityID).Expect().
				Status(http.StatusOK).JSON().Object().Value("id").String().Equal(visibilityID)
		})
	})
})
