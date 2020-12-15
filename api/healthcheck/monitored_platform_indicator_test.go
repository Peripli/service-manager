package healthcheck

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/types"
	storagefakes2 "github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Monitored Platforms Indicator", func() {
	var indicator health.Indicator
	var repository *storagefakes2.FakeStorage
	var ctx context.Context
	var platforms []*types.Platform
	var createPlatform func(string, bool, bool)
	BeforeEach(func() {
		ctx = context.TODO()
		repository = &storagefakes2.FakeStorage{}
		createPlatform = func(name string, active bool, monitored bool) {
			platform := &types.Platform{
				Name: name,
				Type: "kubernetes",
			}
			platform.ID = name
			platform.Active = active
			if !active {
				platform.LastActive = time.Now().Add(-61 * 24 * time.Hour)
			}
			if monitored {
				labels := types.Labels{}
				labels[types.Monitored] = []string{"true"}
				platform.SetLabels(labels)
			}

			platforms = append(platforms, platform)

		}

		indicator = NewMonioredPlatformsIndicator(ctx, repository, 40)

	})

	Context("Name", func() {
		It("should not be empty", func() {
			Expect(indicator.Name()).Should(Equal(MonitoredPlatformsHealthIndicatorName))
		})
	})
	Context("When no platforms are labeled as monitored", func() {
		BeforeEach(func() {
			objectList := &types.Platforms{[]*types.Platform{}}
			repository.QueryForListReturns(objectList, nil)
		})
		It("Healthcheck should be healthy", func() {
			details, err := indicator.Status()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(details).Should(BeEmpty())
		})
	})

	Context("When there are monitored platforms", func() {
		BeforeEach(func() {
			for i := 0; i < 2; i++ {
				createPlatform(fmt.Sprintf("kubernentes-active-%d", i), true, true)
			}
		})
		Context("inactive platform exceed the threshold", func() {
			BeforeEach(func() {
				for i := 0; i < 2; i++ {
					createPlatform(fmt.Sprintf("kubernentes-inactive-%d", i), false, true)
				}
				repository.QueryForListReturns(&types.Platforms{platforms}, nil)
			})
			It("Should return error", func() {
				details, err := indicator.Status()
				detailsH:= details.(map[string]*health.Health)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("50% of the monitored platforms are failing"))
				Expect(len(detailsH)).To(Equal(4))

			})
		})
		Context("inactive platform less than threshold", func() {
			BeforeEach(func() {
				createPlatform("kubernentes-inactive", false, true)
				repository.QueryForListReturns(&types.Platforms{platforms}, nil)
			})
			It("Should return error", func() {
				details, err := indicator.Status()
				detailsH:= details.(map[string]*health.Health)
				Expect(len(detailsH)).To(Equal(5))
				Expect(err).ShouldNot(HaveOccurred())

			})
		})

	})

})
