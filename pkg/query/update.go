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
	"errors"
	"fmt"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/tidwall/gjson"
)

// LabelOperation is an operation to be performed on labels
type LabelOperation string

const (
	// AddLabelOperation takes a label and adds it to the entity's labels
	AddLabelOperation LabelOperation = "add"
	// AddLabelValuesOperation takes a key and values and adds the values to the label with this key
	AddLabelValuesOperation LabelOperation = "add_values"
	// RemoveLabelOperation takes a key and removes the label with this key
	RemoveLabelOperation LabelOperation = "remove"
	// RemoveLabelValuesOperation takes a key and values and removes the values from the label with this key
	RemoveLabelValuesOperation LabelOperation = "remove_values"
)

// RequiresValues returns true if the operation requires values to be provided
func (o LabelOperation) RequiresValues() bool {
	return o != RemoveLabelOperation
}

// LabelChange represents the changes that should be performed to a label
type LabelChange struct {
	Operation LabelOperation `json:"op"`
	Key       string         `json:"key"`
	Values    []string       `json:"values"`
}

func (lc *LabelChange) Validate() error {
	if lc.Operation.RequiresValues() && len(lc.Values) == 0 {
		return fmt.Errorf("operation %s requires values to be provided", lc.Operation)
	}
	if lc.Key == "" || lc.Operation == "" {
		return errors.New("both key and operation are missing but are required for label change")
	}
	return nil
}

type LabelChanges []*LabelChange

func (lc *LabelChanges) Validate() error {
	for _, labelChange := range *lc {
		return labelChange.Validate()
	}
	return nil
}

// LabelChangesFromJSON returns the label changes from the json byte array and an error if the changes are not valid
func LabelChangesFromJSON(jsonBytes []byte) ([]*LabelChange, error) {
	labelChanges := LabelChanges{}
	labelChangesBytes := gjson.GetBytes(jsonBytes, "labels").String()
	if len(labelChangesBytes) <= 0 {
		return LabelChanges{}, nil
	}

	if err := util.BytesToObject([]byte(labelChangesBytes), &labelChanges); err != nil {
		return nil, err
	}

	for _, v := range labelChanges {
		if v.Operation == RemoveLabelOperation {
			v.Values = nil
		}
	}
	return labelChanges, nil
}
