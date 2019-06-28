package storage_test

import (
	"context"
	"testing"

	"github.com/Peripli/service-manager/pkg/query"

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

	BeforeEach(func() {
		ctx = common.NewTestContextBuilder().Build()
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	Context("when resource is created without labels", func() {
		var platform types.Object
		var err error
		BeforeEach(func() {
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

		It("should be fetched from storage without any labels", func() {
			byID := query.ByField(query.EqualsOperator, "id", platform.GetID())
			obj, err := ctx.SMRepository.Get(context.Background(), types.PlatformType, byID)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(obj.GetLabels()).To(HaveLen(0))
		})
	})
})
