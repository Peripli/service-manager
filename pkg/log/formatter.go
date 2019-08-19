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

package log

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type kibanaEntry struct {
	WrittenAt        string        `json:"written_at"`
	WrittenTimestamp string        `json:"written_ts"`
	ComponentType    string        `json:"component_type"`
	CorrelationID    string        `json:"correlation_id"`
	Type             string        `json:"type"`
	Logger           string        `json:"logger"`
	Level            string        `json:"level"`
	Message          string        `json:"msg"`
	Fields           logrus.Fields `json:"-"`
}

// MarshalJSON marshals the kibana entry by inlining the logrus fields instead of being nested in the "Fields" tag
func (k kibanaEntry) MarshalJSON() ([]byte, error) {
	type Entry kibanaEntry
	bytes, err := json.Marshal(Entry(k))
	if err != nil {
		return nil, err
	}

	var result map[string]json.RawMessage
	if err = json.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}

	for k, v := range k.Fields {
		field, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		result[k] = field
	}

	return json.Marshal(result)
}

// KibanaFormatter is a logrus formatter that formats an entry for Kibana
type KibanaFormatter struct {
}

// Format formats a logrus entry for Kibana logging
func (f *KibanaFormatter) Format(e *logrus.Entry) ([]byte, error) {
	componentName, exists := e.Data[FieldComponentName].(string)
	if !exists {
		componentName = "-"
	}
	delete(e.Data, FieldComponentName)

	correlationID, exists := e.Data[FieldCorrelationID].(string)
	if !exists {
		correlationID = "-"
	}
	delete(e.Data, FieldCorrelationID)

	if errorField, exists := e.Data[logrus.ErrorKey].(error); exists {
		e.Message = e.Message + ": " + errorField.Error()
	}
	delete(e.Data, logrus.ErrorKey)

	kibanaEntry := &kibanaEntry{
		Logger:           componentName,
		Level:            e.Level.String(),
		Message:          e.Message,
		CorrelationID:    correlationID,
		Type:             "log",
		ComponentType:    "application",
		WrittenAt:        e.Time.UTC().Format(time.RFC3339Nano),
		WrittenTimestamp: fmt.Sprintf("%d", e.Time.UTC().Unix()),
		Fields:           e.Data,
	}
	serialized, err := json.Marshal(kibanaEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON: %v", err)
	}
	return append(serialized, '\n'), nil
}
