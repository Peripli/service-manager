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
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const (
	// CriteriaFilterName is the name of the label changes filter
	LabelChangesFilterName = "LabelChangeFilter"
)

// LabelChange is filter that configures label changes per request.
type LabelChange struct {
}

// Name implements the web.Filter interface and returns the identifier of the filter.
func (*LabelChange) Name() string {
	return LabelChangesFilterName
}

// Run represents the selection criteria middleware function that processes the request and configures the request-scoped selection criteria.
func (l *LabelChange) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	labelChanges, err := query.BuildLabelChangeForRequestBody(req.Body)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}
	ctx, err = query.AddLabelChanges(ctx, labelChanges...)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}
	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed.
func (*LabelChange) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}
