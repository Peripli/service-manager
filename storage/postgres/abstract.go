/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */
package postgres

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"
)

func updateQuery(tableName string, structure interface{}) (string, error) {
	if !structs.IsStruct(structure) {
		return "", fmt.Errorf("unable to query %s", tableName)
	}
	s := structs.New(structure)
	fields := s.Fields()
	set := make([]string, 0, len(fields))

	for _, field := range fields {
		if field.IsEmbedded() || field.IsZero() {
			continue
		}
		dbTag := field.Tag("db")
		if dbTag == "" {
			dbTag = strings.ToLower(field.Name())
		}
		set = append(set, fmt.Sprintf("%s = :%s", dbTag, dbTag))
	}
	if len(set) == 0 {
		return "", nil
	}
	query := fmt.Sprintf("UPDATE "+tableName+" SET %s WHERE id = :id",
		strings.Join(set, ", "))

	return query, nil
}
