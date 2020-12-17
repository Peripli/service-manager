package storage_test

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
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
		ctx = common.NewTestContextBuilderWithSecurity().Build()
		platform, err = ctx.SMRepository.Create(context.Background(), &types.Platform{
			Base: types.Base{
				ID:    "id",
				Ready: true,
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

	Context("when updating labels only", func() {
		const (
			label1Key    = "label1Key"
			label1Value1 = "label1Value1"
			label1Value2 = "label1Value2"
			label2Key    = "label2Key"
			label2Value1 = "label2Value1"
			label2Value2 = "label2Value2"
			label3Key    = "label3Key"
			label3Value1 = "label3Value1"
			label3Value2 = "label3Value2"
		)

		It("successfully adds new labels, adds and removes new values to existing values and removes existing labels", func() {
			byID := query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err := ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(obj.GetLabels()).To(HaveLen(0))

			By("Successfully adds new labels")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.AddLabelOperation,
					Key:       label1Key,
					Values:    []string{label1Value1},
				},
				{
					Operation: types.AddLabelOperation,
					Key:       label2Key,
					Values:    []string{label2Value1},
				},
				{
					Operation: types.AddLabelOperation,
					Key:       label3Key,
					Values:    []string{label3Value1, label3Value2},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())

			compareLabels(obj.GetLabels(), types.Labels{
				label1Key: []string{label1Value1},
				label2Key: []string{label2Value1},
				label3Key: []string{label3Value1, label3Value2},
			})

			By("Does not fail if label already exists")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.AddLabelOperation,
					Key:       label1Key,
					Values:    []string{label1Value1},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			compareLabels(obj.GetLabels(), types.Labels{
				label1Key: []string{label1Value1},
				label2Key: []string{label2Value1},
				label3Key: []string{label3Value1, label3Value2},
			})

			By("Successfully adds new values to existing labels")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.AddLabelValuesOperation,
					Key:       label1Key,
					Values:    []string{label1Value2},
				},
				{
					Operation: types.AddLabelValuesOperation,
					Key:       label2Key,
					Values:    []string{label2Value2},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			compareLabels(obj.GetLabels(), types.Labels{
				label1Key: []string{label1Value1, label1Value2},
				label2Key: []string{label2Value1, label2Value2},
				label3Key: []string{label3Value1, label3Value2},
			})

			By("Does not fail if label value already exists")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.AddLabelValuesOperation,
					Key:       label1Key,
					Values:    []string{label1Value2},
				},
				{
					Operation: types.AddLabelValuesOperation,
					Key:       label2Key,
					Values:    []string{label2Value2},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			compareLabels(obj.GetLabels(), types.Labels{
				label1Key: []string{label1Value1, label1Value2},
				label2Key: []string{label2Value1, label2Value2},
				label3Key: []string{label3Value1, label3Value2},
			})

			By("Successfully removes existing values from existing labels")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.RemoveLabelValuesOperation,
					Key:       label1Key,
					Values:    []string{label1Value2},
				},
				{
					Operation: types.RemoveLabelValuesOperation,
					Key:       label2Key,
					Values:    []string{label2Value2},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			compareLabels(obj.GetLabels(), types.Labels{
				label1Key: []string{label1Value1},
				label2Key: []string{label2Value1},
				label3Key: []string{label3Value1, label3Value2},
			})

			By("Does not fail if label value does not exist")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.RemoveLabelValuesOperation,
					Key:       label1Key,
					Values:    []string{label1Value2},
				},
				{
					Operation: types.RemoveLabelValuesOperation,
					Key:       label2Key,
					Values:    []string{label2Value2},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			compareLabels(obj.GetLabels(), types.Labels{
				label1Key: []string{label1Value1},
				label2Key: []string{label2Value1},
				label3Key: []string{label3Value1, label3Value2},
			})

			By("Successfully removes existing labels")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.RemoveLabelOperation,
					Key:       label1Key,
				},
				{
					Operation: types.RemoveLabelOperation,
					Key:       label2Key,
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(obj.GetLabels()).To(Equal(types.Labels{
				label3Key: []string{label3Value1, label3Value2},
			}))

			By("Does not fail if label does not exist")
			err = ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, platform.GetID(), types.LabelChanges{
				{
					Operation: types.RemoveLabelOperation,
					Key:       label1Key,
				},
				{
					Operation: types.RemoveLabelOperation,
					Key:       label2Key,
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			byID = query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err = ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			compareLabels(obj.GetLabels(), types.Labels{
				label3Key: []string{label3Value1, label3Value2},
			})
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
				if updatedPlatform, err = storage.Update(ctx, platformFromStorage, types.LabelChanges{}); err != nil {
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

	Context("when plan has no 'free' property", func() {
		var testPlanWithoutFree = `
	{
      "name": "another-paid-plan-name-%[1]s",
      "id": "%[1]s",
      "description": "test-description",
      "bindable": true
    }
`
		var catalogID string

		BeforeEach(func() {
			UUID, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			catalogID = UUID.String()
			plan1 := common.GenerateTestPlanFromTemplate(catalogID, testPlanWithoutFree)
			UUID, err = uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			serviceID := UUID.String()
			service1 := common.GenerateTestServiceWithPlansWithID(serviceID, plan1)
			catalog := common.NewEmptySBCatalog()
			catalog.AddService(service1)
			ctx.RegisterBrokerWithCatalog(catalog)
		})

		It("stored with free = true", func() {
			plans := ctx.SMWithBasic.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_id eq '%s'", catalogID))
			planIsFree := plans.First().Object().Value("free").Raw()
			Expect(planIsFree).To(Equal(true))
		})
	})

	Context("when storing technical platform", func() {
		It("should be created without credentials", func() {
			ctx.SMRepository.Create(context.Background(), &types.Platform{
				Base: types.Base{
					ID: "id_1234",
				},
				Name:      "platform",
				Technical: true,
			})

			obj, err := ctx.SMRepository.Get(context.Background(), types.PlatformType, query.ByField(query.EqualsOperator, "id", "id_1234"))
			Expect(err).NotTo(HaveOccurred())
			platform := obj.(*types.Platform)
			Expect(platform.Credentials).To(BeNil())
			Expect(platform.OldCredentials).To(BeNil())
		})
	})
})

func compareLabels(actual, expected types.Labels) {
	Expect(actual).To(HaveLen(len(expected)))
	for labelKey, labelValues := range actual {
		Expect(labelValues).To(ConsistOf(expected[labelKey]))
	}
}
