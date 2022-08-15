/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package healthcheck

import (
	"context"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/health"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	storagefakes2 "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/storagefakes"
	"time"
)

var _ = Describe("Platforms Indicator", func() {
	var indicator health.Indicator
	var repository *storagefakes2.FakeStorage
	var ctx context.Context
	var platform *types.Platform
	const platformMaxInactive = 60 * 24 * time.Hour

	BeforeEach(func() {
		ctx = context.TODO()
		repository = &storagefakes2.FakeStorage{}
		platform = &types.Platform{
			Name:       "test-platform",
			Type:       "kubernetes",
			Active:     false,
			LastActive: time.Now().Add(-61 * 24 * time.Hour),
		}
		platform.ID = "test-platform"
		indicator = NewPlatformIndicator(ctx, repository, func(p *types.Platform) bool {
			hours := time.Since(p.LastActive).Hours()
			return hours > platformMaxInactive.Hours()
		})
	})

	Context("Name", func() {
		It("should not be empty", func() {
			Expect(indicator.Name()).Should(Equal(health.PlatformsIndicatorName))
		})
	})

	Context("There are inactive platforms longer than max inactive allowed", func() {
		BeforeEach(func() {
			objectList := &types.Platforms{Platforms: []*types.Platform{platform}}
			repository.ListReturns(objectList, nil)
		})
		It("should return error", func() {
			details, err := indicator.Status()
			health := details.(map[string]*health.Health)[platform.Name]
			Expect(err).Should(HaveOccurred())
			Expect(health.Details["since"]).ShouldNot(BeNil())
		})
	})

	Context("There are inactive platforms less than max inactive allowed", func() {
		BeforeEach(func() {
			platform := &types.Platform{
				Name:       "test-platform",
				Type:       "kubernetes",
				Active:     false,
				LastActive: time.Now().Add(-59 * 24 * time.Hour),
			}
			objectList := &types.Platforms{Platforms: []*types.Platform{platform}}
			repository.ListReturns(objectList, nil)
		})
		It("should not return error", func() {
			_, err := indicator.Status()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Storage returns error", func() {
		var expectedErr error
		BeforeEach(func() {
			expectedErr = errors.New("storage err")
			repository.ListReturns(nil, expectedErr)
		})
		It("should return error", func() {
			_, err := indicator.Status()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
		})
	})

	Context("All platforms are active", func() {
		BeforeEach(func() {
			platform.Active = true
			objectList := &types.Platforms{Platforms: []*types.Platform{platform}}
			repository.ListReturns(objectList, nil)
		})
		It("should not return error", func() {
			_, err := indicator.Status()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
