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
	"time"
)

var _ = Describe("Healthcheck Registry", func() {

	var registry *Registry

	BeforeEach(func() {
		registry = NewDefaultRegistry()
	})

	When("Constructing default registry", func() {
		It("Has ping indicator", func() {
			indicators := registry.HealthIndicators
			Expect(indicators).To(ConsistOf(&pingIndicator{}))
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

	When("When configure indicators", func() {
		It("Should configure with default settings if settings not provided", func() {
			newIndicator := &testIndicator{}
			registry.HealthIndicators = append(registry.HealthIndicators, newIndicator)

			registry.ConfigureIndicators()

			Expect(newIndicator.Interval()).To(Equal(DefaultIndicatorSettings().Interval))
			Expect(newIndicator.FailuresTreshold()).To(Equal(DefaultIndicatorSettings().FailuresTreshold))
		})

		It("Should configure with provided settings", func() {
			newIndicator := &testIndicator{}
			registry.HealthIndicators = append(registry.HealthIndicators, newIndicator)

			settings := &IndicatorSettings{
				Interval:         50,
				FailuresTreshold: 2,
			}

			registry.HealthSettings[newIndicator.Name()] = settings

			registry.ConfigureIndicators()

			Expect(newIndicator.Interval()).To(Equal(settings.Interval))
			Expect(newIndicator.FailuresTreshold()).To(Equal(settings.FailuresTreshold))
		})
	})
})

type testIndicator struct {
	settings *IndicatorSettings
}

func (i *testIndicator) Name() string {
	return "test"
}

func (i *testIndicator) Interval() time.Duration {
	return i.settings.Interval
}

func (i *testIndicator) FailuresTreshold() int64 {
	return i.settings.FailuresTreshold
}

func (i *testIndicator) Fatal() bool {
	return i.settings.Fatal
}

func (i *testIndicator) Status() (interface{}, error) {
	return nil, nil
}

func (i *testIndicator) Configure(settings *IndicatorSettings) {
	i.settings = settings
}
