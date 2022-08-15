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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.com/jmoiron/sqlx"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util/slice"
)

type PostgresEntity interface {
	storage.Entity
	TableName() string
	RowsToList(rows *sqlx.Rows) (types.ObjectList, error)
	LabelEntity() PostgresLabel
}

type PostgresLabel interface {
	storage.Label
	LabelsTableName() string
	LabelsPrimaryColumn() string
	ReferenceColumn() string
}

type EntityLabelRowCreator func() EntityLabelRow

type EntityLabelRow interface {
	PostgresEntity
	PostgresLabel
}

func validateLabels(entities []PostgresLabel) error {
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

func rowsToList(rows *sqlx.Rows, rowCreator EntityLabelRowCreator, result types.ObjectList) error {
	entities := make(map[string]types.Object)
	labels := make(map[string]map[string][]string)
	for rows.Next() {
		row := rowCreator()
		if err := rows.StructScan(row); err != nil {
			return err
		}
		entity, ok := entities[row.GetID()]
		if !ok {
			var err error
			entity, err = row.ToObject()
			if err != nil {
				return fmt.Errorf("error converting pg rows to list: %s", err)
			}
			entities[row.GetID()] = entity
			result.Add(entity)
		}
		if row.GetKey() != "" {
			if labels[entity.GetID()] == nil {
				labels[entity.GetID()] = make(map[string][]string)
			}
			labels[entity.GetID()][row.GetKey()] = append(labels[entity.GetID()][row.GetKey()], row.GetValue())
		}
	}
	for i := 0; i < result.Len(); i++ {
		b := result.ItemAt(i)
		b.SetLabels(labels[b.GetID()])
	}
	return nil
}
