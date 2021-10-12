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
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

const PatchOnlyLabelsFilterName = "PatchOnlyLabelsFilter"

// PatchOnlyLabelsFilter checks patch request for service offerings and plans include only label changes
type PatchOnlyLabelsFilter struct {
}

func (*PatchOnlyLabelsFilter) Name() string {
	return PatchOnlyLabelsFilterName
}

func (*PatchOnlyLabelsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	if !(util.LabelsOnly(req)) {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Only labels can be patched for service offerings and plans",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return next.Handle(req)
}

func (*PatchOnlyLabelsFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceOfferingsURL + "/**"),
				web.Methods(http.MethodPatch),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path(web.ServicePlansURL + "/**"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}
