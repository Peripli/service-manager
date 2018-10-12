/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package healthcheck

import (
	"testing"

	"github.com/Peripli/service-manager/pkg/health/healthfakes"

	"github.com/Peripli/service-manager/pkg/health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNewCompositeIndicator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Healthcheck Suite")
}

var _ = Describe("Healthcheck Composite indicator", func() {

	Context("New Composite indicator", func() {
		When("No indicators are provided ", func() {
			It("Has default name", func() {
				indicator := newCompositeIndicator(nil, nil)
				Expect(indicator.Name()).To(Equal(defaultCompositeName))
			})
		})

		When("Has indicators", func() {
			It("Has the name of the indicator", func() {
				fakeIndicator1 := &healthfakes.FakeIndicator{}
				fakeIndicator1.NameReturns("fake1")

				fakeIndicator2 := &healthfakes.FakeIndicator{}
				fakeIndicator2.NameReturns("fake2")
				indicators := []health.Indicator{fakeIndicator1, fakeIndicator2}
				indicator := newCompositeIndicator(indicators, nil)

				Expect(indicator.Name()).To(Equal(aggregateIndicatorNames(indicators)))
			})
		})
	})

	When("Checking health", func() {
		Context("With empty indicators", func() {
			It("Returns unknown status", func() {
				indicator := newCompositeIndicator(nil, &health.DefaultAggregationPolicy{})
				h := indicator.Health()
				Expect(h.Status).To(Equal(health.StatusUnknown))
				Expect(h.Details["error"]).ToNot(BeNil())
			})
		})

		Context("With provided indicators", func() {
			It("Aggregates the healths", func() {
				testIndicator := &healthfakes.FakeIndicator{}
				testIndicator.HealthReturns(health.New().Up())
				testIndicator.NameReturns("fake")

				aggregationPolicy := &healthfakes.FakeAggregationPolicy{}
				defaultAggregationPolicy := &health.DefaultAggregationPolicy{}
				aggregationPolicy.ApplyStub = defaultAggregationPolicy.Apply

				indicator := newCompositeIndicator([]health.Indicator{testIndicator}, aggregationPolicy)
				invocationsCnt := aggregationPolicy.ApplyCallCount()
				health := indicator.Health()
				Expect(aggregationPolicy.ApplyCallCount()).To(Equal(invocationsCnt + 1))
				Expect(health.Details[testIndicator.Name()]).ToNot(BeNil())
			})
		})
	})
})
