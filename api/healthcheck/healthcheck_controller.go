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
	"context"
	gohealth "github.com/InVisionApp/go-health"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/health"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"net/http"
)

// controller platform controller
type controller struct {
	health     gohealth.IHealth
	thresholds map[string]int64
}

// NewController returns a new healthcheck controller with the given health and thresholds
func NewController(health gohealth.IHealth, thresholds map[string]int64) web.Controller {
	return &controller{
		health:     health,
		thresholds: thresholds,
	}
}

// healthCheck handler for GET /v1/monitor/health
func (c *controller) healthCheck(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logger := log.C(ctx)
	logger.Debugf("Performing health check...")
	healthState, _, _ := c.health.State()
	healthResult := c.aggregate(ctx, healthState)
	var status int
	if healthResult.Status == health.StatusUp {
		status = http.StatusOK
	} else {
		status = http.StatusServiceUnavailable
	}
	return util.NewJSONResponse(status, healthResult)
}

func (c *controller) aggregate(ctx context.Context, healthState map[string]gohealth.State) *health.Health {
	if len(healthState) == 0 {
		return health.New().WithStatus(health.StatusUp)
	}

	details := make(map[string]interface{})
	overallStatus := health.StatusUp
	for name, state := range healthState {
		if state.Fatal && state.ContiguousFailures >= c.thresholds[name] {
			overallStatus = health.StatusDown
		}
		state.Status = convertStatus(state.Status)
		if !web.IsAuthorized(ctx) {
			state.Details = nil
			state.Err = ""
		}
		details[name] = state
	}

	return health.New().WithStatus(overallStatus).WithDetails(details)
}

func convertStatus(status string) string {
	switch status {
	case "ok":
		return string(health.StatusUp)
	case "failed":
		return string(health.StatusDown)
	default:
		return string(health.StatusUnknown)
	}
}
