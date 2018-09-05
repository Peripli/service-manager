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
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)


// Controller platform controller
type Controller struct {
	Storage storage.Storage
}

var _ web.Controller = &Controller{}

var statusRunningResponse = map[string]interface{}{
	"status": "UP",
	"storage": map[string]interface{}{
		"status": "UP",
	},
}

var statusStorageFailureResponse = map[string]interface{}{
	"status": "OUT_OF_SERVICE",
	"storage": map[string]interface{}{
		"status": "DOWN",
	},
}

// healthCheck handler for GET /v1/monitor/health
func (c *Controller) healthCheck(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logger := log.C(ctx)
	logger.Debug("Performing health check...")

	if err := c.Storage.Ping(); err != nil {
		logger.Debugf("storage.Ping failed: %s", err)
		return util.NewJSONResponse(http.StatusServiceUnavailable, statusStorageFailureResponse)
	}

	logger.Debug("Successfully completed health check")
	return util.NewJSONResponse(http.StatusOK, statusRunningResponse)
}
