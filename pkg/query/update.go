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
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/tidwall/gjson"
)

// LabelChangesFromJSON returns the label changes from the json byte array and an error if the changes are not valid
func LabelChangesFromJSON(jsonBytes []byte) ([]*types.LabelChange, error) {
	labelChanges := types.LabelChanges{}
	labelChangesBytes := gjson.GetBytes(jsonBytes, "labels").String()
	if len(labelChangesBytes) <= 0 {
		return types.LabelChanges{}, nil
	}

	if err := util.BytesToObject([]byte(labelChangesBytes), &labelChanges); err != nil {
		return nil, err
	}

	return labelChanges, nil
}

// ApplyLabelChangesToLabels applies the specified label changes to the specified labels
func ApplyLabelChangesToLabels(changes types.LabelChanges, labels types.Labels) (types.Labels, types.Labels, types.Labels) {
	mergedLabels, labelsToAdd, labelsToRemove := types.Labels{}, types.Labels{}, types.Labels{}
	for k, v := range labels {
		mergedLabels[k] = v
	}

	for _, change := range changes {
		switch change.Operation {
		case types.AddLabelOperation:
			fallthrough
		case types.AddLabelValuesOperation:
			if len(change.Values) == 0 {
				addValue(mergedLabels, change, "", labelsToAdd)
			}
			for _, value := range change.Values {
				addValue(mergedLabels, change, value, labelsToAdd)
			}
		case types.RemoveLabelOperation:
			fallthrough
		case types.RemoveLabelValuesOperation:
			if len(change.Values) == 0 {
				labelsToRemove[change.Key] = labels[change.Key]
				delete(mergedLabels, change.Key)
			} else {
				labelsToRemove[change.Key] = append(labelsToRemove[change.Key], change.Values...)
				for _, valueToRemove := range change.Values {
					for i, value := range mergedLabels[change.Key] {
						if value == valueToRemove {
							mergedLabels[change.Key] = append(mergedLabels[change.Key][:i], mergedLabels[change.Key][i+1:]...)
							if len(mergedLabels[change.Key]) == 0 {
								delete(mergedLabels, change.Key)
							}
						}
					}
				}
			}
		}
	}

	return mergedLabels, labelsToAdd, labelsToRemove
}

func addValue(mergedLabels types.Labels, change *types.LabelChange, value string, labelsToAdd types.Labels) {
	found := false
	for _, currentValue := range mergedLabels[change.Key] {
		if currentValue == value {
			found = true
			break
		}
	}
	if !found {
		labelsToAdd[change.Key] = append(labelsToAdd[change.Key], value)
		mergedLabels[change.Key] = append(mergedLabels[change.Key], value)
	}
}
