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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthcheck Registry", func() {

	var registry *Registry

	BeforeEach(func() {
		registry = NewDefaultRegistry()
	})

	When("Constructing default registry", func() {
		It("Has empty indicators", func() {
			indicators := registry.HealthIndicators
			Expect(len(indicators)).To(Equal(0))
		})
	})

	When("Register health indicator", func() {
		It("Adds a new indicator", func() {
			preAddIndicators := registry.HealthIndicators

			newIndicator := &testIndicator{}
			registry.HealthIndicators = append(registry.HealthIndicators, newIndicator)
			postAddIndicators := registry.HealthIndicators
			for _, indicator := range preAddIndicators {
				Expect(postAddIndicators).To(ContainElement(indicator))
			}
			Expect(postAddIndicators).To(ContainElement(newIndicator))
		})
	})
})

type testIndicator struct {
}

func (i *testIndicator) Name() string {
	return "test"
}

func (i *testIndicator) Status() (interface{}, error) {
	return nil, nil
}
