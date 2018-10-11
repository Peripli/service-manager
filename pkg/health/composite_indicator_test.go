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
	"sync"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNewCompositeIndicator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Healthcheck Suite")
}

var _ = Describe("Healthcheck Composite Indicator", func() {

	Context("New Composite Indicator", func() {
		It("Uses Default aggregator", func() {
			indicator := NewCompositeIndicator(nil)
			Expect(indicator.(*CompositeIndicator).aggregator).To(BeAssignableToTypeOf(&DefaultAggregator{}))
		})

		It("Has a name", func() {
			indicator := NewCompositeIndicator(nil)
			Expect(indicator.Name()).ToNot(BeEmpty())
		})
	})

	When("Checking health", func() {
		Context("With empty indicators", func() {
			It("Returns unknown status", func() {
				indicator := NewCompositeIndicator(nil)
				health := indicator.Health()
				Expect(health.Status).To(Equal(StatusUnknown))
				Expect(health.Details["error"]).ToNot(BeNil())
			})
		})

		Context("With provided indicators", func() {
			It("Aggregates the healths", func() {
				testIndicator := &testIndicator{}
				indicator := NewCompositeIndicator([]Indicator{testIndicator})
				aggregator := newTestAggregator()
				indicator.(*CompositeIndicator).aggregator = aggregator
				invocationsCnt := aggregator.invocationCount
				health := indicator.Health()
				Expect(aggregator.invocationCount).To(Equal(invocationsCnt + 1))
				Expect(health.Details[testIndicator.Name()]).ToNot(BeNil())
			})
		})
	})
})

type testIndicator struct {
}

func (*testIndicator) Name() string {
	return "test"
}

func (*testIndicator) Health() *Health {
	return New().Up()
}

type testAggregator struct {
	invocationCount int
	delegate        Aggregator
	mutex           sync.Mutex
}

func newTestAggregator() *testAggregator {
	return &testAggregator{
		invocationCount: 0,
		delegate:        &DefaultAggregator{},
		mutex:           sync.Mutex{},
	}
}

func (a *testAggregator) Aggregate(healths map[string]*Health) *Health {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.invocationCount++
	return a.delegate.Aggregate(healths)
}
