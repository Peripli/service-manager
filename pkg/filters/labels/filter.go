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

// ForbiddenLabelOperations describe denied label key operations
type ForbiddenLabelOperations map[string][]query.LabelOperation

func (flo ForbiddenLabelOperations) Validate() error {
	for lk := range flo {
		for _, op := range flo[lk] {
			switch op {
			case query.AddLabelOperation:
				fallthrough
			case query.AddLabelValuesOperation:
				fallthrough
			case query.RemoveLabelOperation:
				fallthrough
			case query.RemoveLabelValuesOperation:
				return nil
			default:
				return fmt.Errorf("label operation %s not recognized", op)
			}
		}
	}
	return nil
}

type ForibiddenLabelOperationsFilter struct {
	forbiddenOperations ForbiddenLabelOperations
}

func NewForbiddenLabelOperationsFilter(forbiddenOperations ForbiddenLabelOperations) *ForibiddenLabelOperationsFilter {
	return &ForibiddenLabelOperationsFilter{
		forbiddenOperations: forbiddenOperations,
	}
}

func (flo *ForibiddenLabelOperationsFilter) Name() string {
	return "ForbiddenLabelOperationsFilter"
}

func (flo *ForibiddenLabelOperationsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	if req.Method == http.MethodPost {
		body := gjson.ParseBytes(req.Body).Map()
		labelsBytes := []byte(body["labels"].String())
		if len(labelsBytes) == 0 {
			return next.Handle(req)
		}

		labels := types.Labels{}
		if err := util.BytesToObject(labelsBytes, &labels); err != nil {
			return nil, err
		}
		for lKey := range labels {
			labelOperations, found := flo.forbiddenOperations[lKey]
			if !found {
				continue
			}
			for _, lo := range labelOperations {
				if lo == query.AddLabelOperation || lo == query.AddLabelValuesOperation {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("Set/Add values for label %s is not allowed", lKey),
						StatusCode:  http.StatusBadRequest,
					}
				}
			}
		}
	} else if req.Method == http.MethodPatch {
		labelChanges, err := query.LabelChangesFromJSON(req.Body)
		if err != nil {
			return nil, err
		}
		for _, lc := range labelChanges {
			deniedLabelOperations, found := flo.forbiddenOperations[lc.Key]
			if !found {
				continue
			}
			for _, op := range deniedLabelOperations {
				if op == lc.Operation {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("Operation %s is not allowed for label %s", lc.Operation, lc.Key),
						StatusCode:  http.StatusBadRequest,
					}
				}
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
