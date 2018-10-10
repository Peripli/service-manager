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

// CompositeIndicator aggregates multiple health indicators and provides one detailed health
type CompositeIndicator struct {
	aggregator Aggregator
	indicators []Indicator
}

// NewCompositeIndicator returns a new CompositeIndicator for the provided health indicators
func NewCompositeIndicator(indicators []Indicator) Indicator {
	return &CompositeIndicator{
		aggregator: &DefaultAggregator{},
		indicators: indicators,
	}
}

// Name returns the name of the CompositeIndicator
func (i *CompositeIndicator) Name() string {
	return "CompositeHealthIndicator"
}

// Health returns the aggregated health of all health indicators
func (i *CompositeIndicator) Health() *Health {
	healths := make(map[string]*Health)
	for _, indicator := range i.indicators {
		healths[indicator.Name()] = indicator.Health()
	}
	return i.aggregator.Aggregate(healths)
}
