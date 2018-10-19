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

package health

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthcheck Registry", func() {

	var registry Registry

	BeforeEach(func() {
		registry = NewDefaultRegistry()
	})

	When("Constructing default registry", func() {
		It("Has ping indicator and default aggregation policy", func() {
			indicators := registry.HealthIndicators()
			Expect(indicators).To(ConsistOf(&pingIndicator{}))

			policy := registry.HealthAggregationPolicy()
			Expect(policy).To(BeAssignableToTypeOf(&DefaultAggregationPolicy{}))
		})
	})

	Context("Register aggregation policy", func() {
		It("Overrides the previous", func() {
			policy := registry.HealthAggregationPolicy()
			Expect(policy).To(BeAssignableToTypeOf(&DefaultAggregationPolicy{}))

			registry.RegisterHealthAggregationPolicy(&testAggregationPolicy{})
			policy = registry.HealthAggregationPolicy()
			Expect(policy).To(BeAssignableToTypeOf(&testAggregationPolicy{}))
		})
	})

	Context("Register health indicator", func() {
		It("Adds a new indicator", func() {
			preAddIndicators := registry.HealthIndicators()

			newIndicator := &testIndicator{}
			registry.AddHealthIndicator(newIndicator)
			postAddIndicators := registry.HealthIndicators()
			for _, indicator := range preAddIndicators {
				Expect(postAddIndicators).To(ContainElement(indicator))
			}
			Expect(postAddIndicators).To(ContainElement(newIndicator))
		})
	})
})

type testAggregationPolicy struct {
}

func (*testAggregationPolicy) Apply(healths map[string]*Health) *Health {
	return New().Up()
}

type testIndicator struct {
}

func (*testIndicator) Name() string {
	return "test"
}

func (*testIndicator) Health() *Health {
	return New().Up()
}
