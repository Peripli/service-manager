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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const (
	// CriteriaFilterName is the name of the criteria filter
	CriteriaFilterName = "CriteriaFilter"
)

// SelectionCriteria is filter that configures selection criteria per request.
type SelectionCriteria struct {
}

// Name implements the web.Filter interface and returns the identifier of the filter.
func (*SelectionCriteria) Name() string {
	return CriteriaFilterName
}

// Run represents the selection criteria middleware function that processes the request and configures the request-scoped selection criteria.
func (l *SelectionCriteria) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	var criteria []query.Criterion
	for _, queryType := range query.CriteriaTypes {
		queryValue := req.URL.Query().Get(string(queryType))
		queryCriteria, err := query.Parse(queryType, queryValue)
		if err != nil {
			return nil, err
		}
		criteria = append(criteria, queryCriteria...)
	}
	ctx, err := query.AddCriteria(ctx, criteria...)
	if err != nil {
		return nil, err
	}
	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed.
func (*SelectionCriteria) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodGet, http.MethodDelete),
			},
		},
	}
}
