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

package health

// NewDefaultRegistry returns a default health registry with a single ping indicator and a default aggregation policy
func NewDefaultRegistry() Registry {
	return &defaultRegistry{
		indicators:        []Indicator{&pingIndicator{}},
		aggregationPolicy: &DefaultAggregationPolicy{},
	}
}

type defaultRegistry struct {
	indicators        []Indicator
	aggregationPolicy AggregationPolicy
}

func (p *defaultRegistry) RegisterHealthAggregationPolicy(aggregator AggregationPolicy) {
	p.aggregationPolicy = aggregator
}

func (p *defaultRegistry) HealthAggregationPolicy() AggregationPolicy {
	return p.aggregationPolicy
}

func (p *defaultRegistry) AddHealthIndicator(indicator Indicator) {
	p.indicators = append(p.indicators, indicator)
}

func (p *defaultRegistry) HealthIndicators() []Indicator {
	return p.indicators
}
