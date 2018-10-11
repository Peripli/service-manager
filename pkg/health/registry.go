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

import "sync"

// DefaultRegistry returns a default health indicator registry with a single ping indicator and a default aggregationPolicy
func DefaultRegistry() Registry {
	return &defaultRegistry{
		indicators:        []Indicator{&pingIndicator{}},
		aggregationPolicy: &DefaultAggregationPolicy{},
		indicatorsMutex:   sync.Mutex{},
		aggregatorMutex:   sync.Mutex{},
	}
}

type defaultRegistry struct {
	indicators        []Indicator
	aggregationPolicy AggregationPolicy
	indicatorsMutex   sync.Mutex
	aggregatorMutex   sync.Mutex
}

func (p *defaultRegistry) RegisterHealthAggregationPolicy(aggregator AggregationPolicy) {
	p.aggregatorMutex.Lock()
	defer p.aggregatorMutex.Unlock()
	p.aggregationPolicy = aggregator
}

func (p *defaultRegistry) HealthAggregationPolicy() AggregationPolicy {
	return p.aggregationPolicy
}

func (p *defaultRegistry) AddHealthIndicator(indicator Indicator) {
	p.indicatorsMutex.Lock()
	defer p.indicatorsMutex.Unlock()
	p.indicators = append(p.indicators, indicator)
}

func (p *defaultRegistry) HealthIndicators() []Indicator {
	return p.indicators
}
