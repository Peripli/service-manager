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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const ProtectedLabelsFilterName = "ProtectedLabelsFilter"

// ProtectedLabelsFilter checks for forbidden labels being modified/added
type ProtectedLabelsFilter struct {
	protectedLabels map[string]bool
}

// NewProtectedLabelsFilter creates new filter for forbidden labels
func NewProtectedLabelsFilter(forbiddenLabels []string) *ProtectedLabelsFilter {
	forbiddenLabelsMap := make(map[string]bool)
	for _, label := range forbiddenLabels {
		forbiddenLabelsMap[label] = true
	}
	return &ProtectedLabelsFilter{
		protectedLabels: forbiddenLabelsMap,
	}
}

func (flo *ProtectedLabelsFilter) Name() string {
	return ProtectedLabelsFilterName
}

func (flo *ProtectedLabelsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	if len(flo.protectedLabels) == 0 {
		return next.Handle(req)
	}

	if req.Method == http.MethodPost {
		isJSON, err := util.IsJSONContentType(req.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}

		if !isJSON {
			return next.Handle(req)
		}

		var result types.Base
		if err := json.Unmarshal(req.Body, &result); err != nil {
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("Invalid JSON body"),
				StatusCode:  http.StatusBadRequest,
			}
		}

		labels := result.Labels
		for lKey := range labels {
			_, found := flo.protectedLabels[lKey]
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
			_, found := flo.protectedLabels[lc.Key]
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

func (flo *ProtectedLabelsFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
	}
}
