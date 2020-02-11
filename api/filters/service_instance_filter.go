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
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
)

const platformIDProperty = "platform_id"

const ServiceInstanceFilterName = "ServiceInstanceFilter"

// ServiceInstanceFilter ensures that if a platform is provided for provisioning request that it's the SM Platform.
// It also limits Patch and Delete requests to instances created in the SM platform.
type ServiceInstanceFilter struct {
}

func (*ServiceInstanceFilter) Name() string {
	return ServiceInstanceFilterName
}

func (*ServiceInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	reqCtx := req.Context()
	platformID := gjson.GetBytes(req.Body, platformIDProperty).Str

	if platformID != "" && platformID != types.SMPlatform {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Providing %s property during provisioning/updating of a service instance is forbidden", platformIDProperty),
			StatusCode:  http.StatusBadRequest,
		}
	}

	var err error

	switch req.Request.Method {
	case http.MethodPost:
		if platformID == "" {
			req.Body, err = sjson.SetBytes(req.Body, platformIDProperty, types.SMPlatform)
			if err != nil {
				return nil, err
			}
		}
	case http.MethodPatch:
		fallthrough
	case http.MethodDelete:
		byPlatformID := query.ByField(query.EqualsOperator, platformIDProperty, types.SMPlatform)
		ctx, err := query.AddCriteria(reqCtx, byPlatformID)
		if err != nil {
			return nil, err
		}
		req.Request = req.WithContext(ctx)
	}

	return next.Handle(req)
}

func (*ServiceInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodDelete),
			},
		},
	}
}
