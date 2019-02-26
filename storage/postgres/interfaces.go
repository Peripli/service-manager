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

package postgres

import (
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/jmoiron/sqlx"
)

var (
	knownEntities = make(map[types.ObjectType]Entity)
)

func RegisterEntity(objectType types.ObjectType, entity Entity) {
	if _, exists := knownEntities[objectType]; exists {
		panic(fmt.Sprintf("object type %s is already associated with a postgesql entity type", objectType))
	}
	knownEntities[objectType] = entity
}

func validateLabels(entities []Label) error {
	pairs := make(map[string][]string)
	for _, bl := range entities {
		newKey := bl.GetKey()
		newValue := bl.GetValue()
		val, exists := pairs[newKey]
		if exists && slice.StringsAnyEquals(val, newValue) {
			return fmt.Errorf("duplicate label with key %s and value %s", newKey, newValue)
		}
		pairs[newKey] = append(pairs[newKey], newValue)
	}
	return nil
}

type Identifiable interface {
	GetID() string
}

type PrimaryEntity interface {
	TableName() string
	PrimaryColumn() string
}

type SecondaryEntity interface {
	PrimaryEntity
	ReferenceColumn() string
}

type Entity interface {
	Identifiable
	PrimaryEntity
	Empty() Entity
	RowsToList(rows *sqlx.Rows) (types.ObjectList, error)
	Labels() EntityLabels
	ToObject() types.Object
	FromObject(object types.Object) Entity
}

type Label interface {
	SecondaryEntity
	Empty() Label
	New(entityID, id, key, value string) Label
	GetKey() string
	GetValue() string
}

type EntityLabels interface {
	Single() Label
	ToDTO() types.Labels
	FromDTO(entityID string, labels types.Labels) ([]Label, error)
}
