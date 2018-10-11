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
	"testing"
)

func TestHealthcheck(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Healthcheck Suite")
}

var _ = Describe("Healthcheck Aggregator", func() {

	healthAggregator := &DefaultAggregator{}
	var healths map[string]*Health

	BeforeEach(func() {
		healths = map[string]*Health{
			"test1": New().Up(),
			"test2": New().Up(),
		}
	})

	When("At least one health is DOWN", func() {
		It("Returns DOWN", func() {
			healths["test3"] = New().Down()
			aggregatedHealth := healthAggregator.Aggregate(healths)
			Expect(aggregatedHealth.Status).To(Equal(StatusDown))
		})
	})

	When("All healths are UP", func() {
		It("Returns UP", func() {
			aggregatedHealth := healthAggregator.Aggregate(healths)
			Expect(aggregatedHealth.Status).To(Equal(StatusUp))
		})
	})

	When("Aggregating healths", func() {
		It("Includes them as overall details", func() {
			aggregatedHealth := healthAggregator.Aggregate(healths)
			for name, h := range healths {
				Expect(aggregatedHealth.Details[name]).To(Equal(h))
			}
		})
	})
})
