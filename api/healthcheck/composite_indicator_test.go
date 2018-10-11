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
	"sync"
	"testing"

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
		It("Has a name", func() {
			indicator := newCompositeIndicator(nil, nil)
			Expect(indicator.Name()).ToNot(BeEmpty())
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
				testIndicator := &testIndicator{}
				indicator := newCompositeIndicator([]health.Indicator{testIndicator}, &health.DefaultAggregationPolicy{})
				aggregationPolicy := newTestAggregationPolicy()
				indicator.(*compositeIndicator).aggregationPolicy = aggregationPolicy
				invocationsCnt := aggregationPolicy.invocationCount
				health := indicator.Health()
				Expect(aggregationPolicy.invocationCount).To(Equal(invocationsCnt + 1))
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

func (*testIndicator) Health() *health.Health {
	return health.New().Up()
}

type testAggregationPolicy struct {
	invocationCount int
	delegate        health.AggregationPolicy
	mutex           sync.Mutex
}

func newTestAggregationPolicy() *testAggregationPolicy {
	return &testAggregationPolicy{
		invocationCount: 0,
		delegate:        &health.DefaultAggregationPolicy{},
		mutex:           sync.Mutex{},
	}
}

func (a *testAggregationPolicy) Apply(healths map[string]*health.Health) *health.Health {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.invocationCount++
	return a.delegate.Apply(healths)
}
