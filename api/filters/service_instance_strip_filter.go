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
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

const ServiceInstanceStripFilterName = "ServiceInstanceStripFilter"

var serviceInstanceUnmodifiableProperties = []string{
	"ready", "usable", "context",
}

// ServiceInstanceStripFilter checks post/patch request body for unmodifiable properties
type ServiceInstanceStripFilter struct {
}

func (*ServiceInstanceStripFilter) Name() string {
	return ServiceInstanceStripFilterName
}

func (*ServiceInstanceStripFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	if gjson.GetBytes(req.Body, "id").Exists() {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Invalid request body - providing specific resource id is forbidden",
			StatusCode:  http.StatusBadRequest,
		}
	}

	var err error
	req.Body, err = removePropertiesFromRequest(req.Context(), req.Body, serviceInstanceUnmodifiableProperties)
	if err != nil {
		return nil, err
	}
	return next.Handle(req)
}

func (*ServiceInstanceStripFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
	}
}

func removePropertiesFromRequest(ctx context.Context, body []byte, props []string) ([]byte, error) {
	var err error
	for _, prop := range props {
		for gjson.GetBytes(body, prop).Exists() {
			body, err = sjson.DeleteBytes(body, prop)
			if err != nil {
				log.C(ctx).Errorf("Could not remove %s from body %s", prop, err)
				return nil, &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: "Invalid request body",
					StatusCode:  http.StatusBadRequest,
				}
			}
		}
	}

	return body, nil
}
