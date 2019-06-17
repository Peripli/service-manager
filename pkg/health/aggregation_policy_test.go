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
	"github.com/InVisionApp/go-health"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthcheck AggregationPolicy", func() {

	aggregationPolicy := &DefaultAggregationPolicy{}
	var healths map[string]health.State
	var failuresTreshold int64 = 3

	BeforeEach(func() {
		healths = map[string]health.State{
			"test1": {Status: "ok"},
			"test2": {Status: "ok"},
		}
	})

	When("No healths are provided", func() {
		It("Returns UNKNOWN and an error detail", func() {
			aggregatedHealth := aggregationPolicy.Apply(nil, failuresTreshold)
			Expect(aggregatedHealth.Status).To(Equal(StatusUnknown))
			Expect(aggregatedHealth.Details["error"]).ToNot(BeNil())
		})
	})

	When("At least one health is DOWN", func() {
		It("Returns DOWN", func() {
			healths["test3"] = health.State{Status: "failed", ContiguousFailures: 4}
			aggregatedHealth := aggregationPolicy.Apply(healths, failuresTreshold)
			Expect(aggregatedHealth.Status).To(Equal(StatusDown))
		})
	})

	When("There is DOWN healths but not more than treshold times in a row", func() {
		It("Returns UP", func() {
			healths["test3"] = health.State{Status: "failed"}
			aggregatedHealth := aggregationPolicy.Apply(healths, failuresTreshold)
			Expect(aggregatedHealth.Status).To(Equal(StatusUp))
		})
	})

	When("All healths are UP", func() {
		It("Returns UP", func() {
			aggregatedHealth := aggregationPolicy.Apply(healths, failuresTreshold)
			Expect(aggregatedHealth.Status).To(Equal(StatusUp))
		})
	})

	When("Aggregating healths", func() {
		It("Includes them as overall details", func() {
			aggregatedHealth := aggregationPolicy.Apply(healths, failuresTreshold)
			for name, h := range healths {
				Expect(aggregatedHealth.Details[name]).To(Equal(convertStatus(h.Status)))
			}
		})
	})
})
