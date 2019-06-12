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

package labels

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
)

// ForibiddenLabelOperationsFilter checks for forbidden labels being modified/added
type ForibiddenLabelOperationsFilter struct {
	forbiddenLabels map[string]bool
}

// NewForbiddenLabelOperationsFilter creates new filter for forbidden labels
func NewForbiddenLabelOperationsFilter(forbiddenLabels []string) *ForibiddenLabelOperationsFilter {
	forbiddenLabelsMap := make(map[string]bool)
	for _, label := range forbiddenLabels {
		forbiddenLabelsMap[label] = true
	}
	return &ForibiddenLabelOperationsFilter{
		forbiddenLabels: forbiddenLabelsMap,
	}
}

func (flo *ForibiddenLabelOperationsFilter) Name() string {
	return "ForbiddenLabelOperationsFilter"
}

func (flo *ForibiddenLabelOperationsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	if len(flo.forbiddenLabels) == 0 {
		return next.Handle(req)
	}

	if req.Method == http.MethodPost {
		body := gjson.ParseBytes(req.Body).Map()
		if _, found := body["labels"]; !found {
			return next.Handle(req)
		}

		labelsBytes := []byte(body["labels"].String())
		if len(labelsBytes) == 0 {
			return next.Handle(req)
		}

		labels := types.Labels{}
		if err := util.BytesToObject(labelsBytes, &labels); err != nil {
			return nil, err
		}
		for lKey := range labels {
			_, found := flo.forbiddenLabels[lKey]
			if !found {
				continue
			}

			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("Set/Add values for label %s is not allowed", lKey),
				StatusCode:  http.StatusBadRequest,
			}
		}
	} else if req.Method == http.MethodPatch {
		labelChanges, err := query.LabelChangesFromJSON(req.Body)
		if err != nil {
			return nil, err
		}
		for _, lc := range labelChanges {
			_, found := flo.forbiddenLabels[lc.Key]
			if !found {
				continue
			}

			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("Modifying is not allowed for label %s", lc.Key),
				StatusCode:  http.StatusBadRequest,
			}
		}
	}
	return next.Handle(req)
}

func (flo *ForibiddenLabelOperationsFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBrokersURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path(web.PlatformsURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
	}
}
