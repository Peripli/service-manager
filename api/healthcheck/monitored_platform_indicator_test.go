package healthcheck

import (
	"context"
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/health"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	storagefakes "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/storagefakes"
	"time"
)

var _ = Describe("Monitored Platforms Indicator", func() {
	var indicator health.Indicator
	var repository *storagefakes.FakeStorage
	var ctx context.Context
	var platforms []*types.Platform
	var createPlatform func(string, bool, bool)
	BeforeEach(func() {
		ctx = context.TODO()
		repository = &storagefakes.FakeStorage{}
		createPlatform = func(name string, active bool, monitored bool) {
			platform := &types.Platform{
				Name: name,
				Type: "kubernetes",
			}
			platform.ID = name
			platform.Active = active
			platform.LastActive = time.Now().Add(20 * time.Second)
			if monitored {
				labels := types.Labels{}
				labels[types.Monitored] = []string{"true"}
				platform.SetLabels(labels)
			}

			platforms = append(platforms, platform)

		}

		indicator = NewMonitoredPlatformsIndicator(ctx, repository, 40)

	})

	Context("Name", func() {
		It("should not be empty", func() {
			Expect(indicator.Name()).Should(Equal(health.MonitoredPlatformsHealthIndicatorName))
		})
	})
	Context("When no platforms are labeled as monitored", func() {
		BeforeEach(func() {
			objectList := &types.Platforms{Platforms: []*types.Platform{}}
			repository.QueryForListReturns(objectList, nil)
		})
		It("Healthcheck should be healthy", func() {
			details, err := indicator.Status()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(details).Should(BeEmpty())
		})
	})

	Context("Storage returns error", func() {
		var expectedErr error
		BeforeEach(func() {
			expectedErr = errors.New("storage err")
			repository.QueryForListReturns(nil, expectedErr)
		})
		It("should return error", func() {
			_, err := indicator.Status()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
		})
	})

	Context("When there are monitored platforms", func() {
		BeforeEach(func() {
			for i := 0; i < 2; i++ {
				createPlatform(fmt.Sprintf("kubernentes-active-%d", i), true, true)
			}
		})
		AfterEach(func() {
			//clear array
			platforms = platforms[:0]
		})
		Context("all platforms are active", func() {
			BeforeEach(func() {
				repository.QueryForListReturns(&types.Platforms{Platforms: platforms}, nil)
			})
			It("should not return an error", func() {
				details, err := indicator.Status()
				detailsH := details.(map[string]*health.Health)
				Expect(len(detailsH)).To(Equal(2))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(detailsH[platforms[0].Name]).NotTo(BeNil())
				Expect(detailsH[platforms[0].Name].Status).To(BeEquivalentTo("UP"))
				Expect(detailsH[platforms[1].Name]).NotTo(BeNil())
				Expect(detailsH[platforms[1].Name].Status).To(BeEquivalentTo("UP"))
			})

		})
		Context("inactive platform exceed the threshold", func() {
			BeforeEach(func() {
				for i := 0; i < 2; i++ {
					createPlatform(fmt.Sprintf("kubernentes-inactive-%d", i), false, true)
				}
				repository.QueryForListReturns(&types.Platforms{Platforms: platforms}, nil)
			})
			It("Should return error", func() {
				details, err := indicator.Status()
				detailsH := details.(map[string]*health.Health)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("50% of the monitored platforms are failing"))
				Expect(len(detailsH)).To(Equal(4))

			})
		})
		Context("inactive platform less than threshold", func() {
			BeforeEach(func() {
				createPlatform("kubernentes-inactive", false, true)
				repository.QueryForListReturns(&types.Platforms{Platforms: platforms}, nil)
			})
			It("Should not return error", func() {
				details, err := indicator.Status()
				Expect(err).ShouldNot(HaveOccurred())
				detailsH := details.(map[string]*health.Health)
				Expect(len(detailsH)).To(Equal(3))
			})
		})

	})

})
