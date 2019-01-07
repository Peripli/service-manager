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

package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/query"
)

type Label struct {
	ID        string    `json:"-"`
	Key       string    `json:"key,omitempty"`
	Value     []string  `json:"value,omitempty"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (l Label) Validate() error {
	if strings.ContainsRune(l.Key, query.Separator) {
		return fmt.Errorf("label key \"%s\" cannot contain whitespaces and special symbol %c", l.Key, query.Separator)
	}
	for _, val := range l.Value {
		if strings.ContainsRune(val, '\n') {
			return fmt.Errorf("label with key \"%s\" has value \"%s\" contaning forbidden new line character", l.Key, val)
		}
	}
	return nil
}
