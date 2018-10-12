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
	"strings"

	"github.com/Peripli/service-manager/pkg/health"
)

const defaultCompositeName = "composite"

// compositeIndicator aggregates multiple health indicators and provides one detailed health
type compositeIndicator struct {
	aggregationPolicy health.AggregationPolicy
	indicators        []health.Indicator
}

// newCompositeIndicator returns a new compositeIndicator for the provided health indicators
func newCompositeIndicator(indicators []health.Indicator, aggregator health.AggregationPolicy) health.Indicator {
	return &compositeIndicator{
		aggregationPolicy: aggregator,
		indicators:        indicators,
	}
}

// Name returns the name of the compositeIndicator
func (i *compositeIndicator) Name() string {
	if len(i.indicators) == 0 {
		return defaultCompositeName
	}
	return aggregateIndicatorNames(i.indicators)
}

// Health returns the aggregated health of all health indicators
func (i *compositeIndicator) Health() *health.Health {
	healths := make(map[string]*health.Health)
	for _, indicator := range i.indicators {
		healths[indicator.Name()] = indicator.Health()
	}
	return i.aggregationPolicy.Apply(healths)
}

func aggregateIndicatorNames(indicators []health.Indicator) string {
	indicatorsCnt := len(indicators)
	builder := strings.Builder{}
	for index, indicator := range indicators {
		builder.WriteString(indicator.Name())
		if index < indicatorsCnt-1 {
			builder.WriteString(", ")
		}
	}
	return builder.String()
}
