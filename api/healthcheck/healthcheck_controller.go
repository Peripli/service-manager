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

package healthcheck

import (
	"github.com/Peripli/service-manager/pkg/util"
	"net/http"

	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

// controller platform controller
type controller struct {
	indicator health.Indicator
}

// NewController returns a new healthcheck controller with the given indicators and aggregation policy
func NewController(indicators []health.Indicator, aggregator health.AggregationPolicy) web.Controller {
	return &controller{
		indicator: newCompositeIndicator(indicators, aggregator),
	}
}

// healthCheck handler for GET /v1/monitor/health
func (c *controller) healthCheck(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logger := log.C(ctx)
	logger.Debugf("Performing health check with %s...", c.indicator.Name())
	healthResult := c.indicator.Health()
	var status int
	if healthResult.Status == health.StatusUp {
		status = http.StatusOK
	} else {
		status = http.StatusServiceUnavailable
	}
	return util.NewJSONResponse(status, healthResult)
}
