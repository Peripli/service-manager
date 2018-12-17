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
	"encoding/json"
	"fmt"

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

// LabelChangeError is an error that shows that the constructed label change cannot be executed
type LabelChangeError struct {
	Message string
}

func (l LabelChangeError) Error() string {
	return l.Message
}

// LabelChange represents the changes that should be performed to a label
type LabelChange struct {
	Operation LabelOperation `json:"op"`
	Key       string         `json:"key"`
	Values    []string       `json:"values"`
}

func (lc LabelChange) Validate() error {
	if lc.Operation.RequiresValues() && len(lc.Values) == 0 {
		return &LabelChangeError{fmt.Sprintf("operation %s requires values to be provided", lc.Operation)}
	}
	if lc.Key == "" || lc.Operation == "" {
		return &LabelChangeError{Message: "both key and operation are required for label change"}
	}
	return nil
}

// LabelChangesFromJSON returns the label changes from the json byte array and an error if the changes are not valid
func LabelChangesFromJSON(jsonBytes []byte) ([]*LabelChange, error) {
	var labelChanges []*LabelChange
	labelChangesBytes := gjson.GetBytes(jsonBytes, "labels").String()
	if len(labelChangesBytes) <= 0 {
		return []*LabelChange{}, nil
	}
	if err := json.Unmarshal([]byte(labelChangesBytes), &labelChanges); err != nil {
		return nil, err
	}
	for _, v := range labelChanges {
		if v.Operation == RemoveLabelOperation {
			v.Values = nil
		}
	}
	for _, change := range labelChanges {
		if err := change.Validate(); err != nil {
			return nil, err
		}
	}
	return labelChanges, nil
}
