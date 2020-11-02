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
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

const (
	PlatformTerminationFilterName = "PlatformTerminationFilter"
)

func NewPlatformTerminationFilter(repository storage.Repository) *platformTerminationFilter {
	return &platformTerminationFilter{
		repository: repository,
	}
}

// platformTerminationFilter ensures that platform provided is considered as not active and only then deletion is possible.
type platformTerminationFilter struct {
	repository storage.Repository
}

func (*platformTerminationFilter) Name() string {
	return PlatformTerminationFilterName
}

func (f *platformTerminationFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	platformID := req.PathParams[web.PathParamResourceID]

	if req.Request.Method == http.MethodDelete && platformID != "" {
		ctx := req.Context()
		byID := query.ByField(query.EqualsOperator, "id", platformID)
		platformObject, err := f.repository.Get(ctx, types.PlatformType, byID)
		if err != nil {
			return nil, util.HandleStorageError(err, types.PlatformType.String())
		}
		platform := platformObject.(*types.Platform)
		if platform.Active {
			return nil, &util.HTTPError{
				ErrorType:   "UnprocessableEntity",
				Description: "Active platform cannot be deleted",
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}
	}

	return next.Handle(req)
}

func (*platformTerminationFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.PlatformsURL + "/**"),
				web.Methods(http.MethodDelete),
			},
		},
	}
}
