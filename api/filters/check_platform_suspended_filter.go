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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const CheckPlatformSuspendedFilterName = "CheckPlatformSuspendedFilter"

// ServiceBindingStripFilter checks post request body for unmodifiable properties
type CheckPlatformSuspendedFilter struct {
}

func (*CheckPlatformSuspendedFilter) Name() string {
	return CheckPlatformSuspendedFilterName
}

func (f *CheckPlatformSuspendedFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	user, ok := web.UserFromContext(req.Context())
	if !ok {
		return next.Handle(req)
	}
	platform := &types.Platform{}
	err := user.Data(platform)
	if err != nil {
		return nil, err
	}
	if platform.Suspended {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "platform suspended",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return next.Handle(req)
}

func (*CheckPlatformSuspendedFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPut, http.MethodPatch),
			},
		},
	}
}
