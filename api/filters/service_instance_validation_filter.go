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
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
)

const platformIDProperty = "platform_id"

const ServiceInstanceValidationFilterName = "ServiceInstanceValidationFilter"

// ServiceInstanceValidationFilter ensures that if a platform is provided for service instance that it's
// the SM Platform and also discards the properties 'ready' and 'usable' if they're provided as these properties
// are only maintained internally.
type ServiceInstanceValidationFilter struct {
}

func (*ServiceInstanceValidationFilter) Name() string {
	return ServiceInstanceValidationFilterName
}

func (*ServiceInstanceValidationFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	platformID := gjson.GetBytes(req.Body, platformIDProperty).Str

	if platformID != "" && platformID != types.SMPlatform {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Providing %s property during provisioning/updating of a service instance is forbidden", platformIDProperty),
			StatusCode:  http.StatusBadRequest,
		}
	}

	var err error
	if req.Method == http.MethodPost && platformID == "" {
		req.Body, err = sjson.SetBytes(req.Body, platformIDProperty, types.SMPlatform)
		if err != nil {
			return nil, err
		}
	}

	req.Body, err = sjson.DeleteBytes(req.Body, "ready")
	if err != nil {
		return nil, err
	}

	req.Body, err = sjson.DeleteBytes(req.Body, "usable")
	if err != nil {
		return nil, err
	}

	return next.Handle(req)
}

func (*ServiceInstanceValidationFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
	}
}
