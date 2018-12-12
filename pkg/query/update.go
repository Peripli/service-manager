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

package query

import (
	"context"
	"encoding/json"

	"github.com/tidwall/gjson"
)

type Operation string

const (
	AddLabelOperation          Operation = "add"
	AddLabelValuesOperation    Operation = "add_values"
	RemoveLabelOperation       Operation = "remove"
	RemoveLabelValuesOperation Operation = "remove_values"
)

type LabelChange struct {
	Operation Operation `json:"op"`
	Key       string    `json:"key"`
	Values    []string  `json:"values"`
}

func (lc LabelChange) Validate() error {
	return nil
}

func BuildLabelChangeForRequestBody(requestBody []byte) ([]LabelChange, error) {
	var labelChanges []LabelChange
	labelChangesBytes := gjson.GetBytes(requestBody, "labels").String()
	if err := json.Unmarshal([]byte(labelChangesBytes), &labelChanges); err != nil {
		return nil, err
	}
	for _, change := range labelChanges {
		if err := change.Validate(); err != nil {
			return nil, err
		}
	}
	return labelChanges, nil
}

type labelChangeCtxKey struct{}

func AddLabelChanges(ctx context.Context, changes ...LabelChange) (context.Context, error) {
	labelChanges := LabelChangesForContext(ctx)
	labelChanges = append(labelChanges, changes...)
	return context.WithValue(ctx, labelChangeCtxKey{}, labelChanges), nil
}

func LabelChangesForContext(ctx context.Context) []LabelChange {
	currentCriteria := ctx.Value(labelChangeCtxKey{})
	if currentCriteria == nil {
		return []LabelChange{}
	}
	return currentCriteria.([]LabelChange)
}
