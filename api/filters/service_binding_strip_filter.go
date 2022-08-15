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

package filters

import (
	"github.com/tidwall/gjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ServiceBindingStripFilterName = "ServiceBindingStripFilter"

var serviceBindingUnmodifiableProperties = []string{
	"credentials", "syslog_drain_url", "route_service_url", "volume_mounts", "endpoints", "ready", "context",
}

// ServiceBindingStripFilter checks post request body for unmodifiable properties
type ServiceBindingStripFilter struct {
}

func (*ServiceBindingStripFilter) Name() string {
	return ServiceBindingStripFilterName
}

func (*ServiceBindingStripFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	if gjson.GetBytes(req.Body, "id").Exists() {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Invalid request body - providing specific resource id is forbidden",
			StatusCode:  http.StatusBadRequest,
		}
	}

	var err error
	req.Body, err = removePropertiesFromRequest(req.Context(), req.Body, serviceBindingUnmodifiableProperties)
	if err != nil {
		return nil, err
	}
	return next.Handle(req)
}

func (*ServiceBindingStripFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBindingsURL + "/**"),
				web.Methods(http.MethodPost),
			},
		},
	}
}
