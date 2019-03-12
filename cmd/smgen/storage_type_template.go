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

package main

const STORAGE_TYPE_TEMPLATE = `
// GENERATED. DO NOT MODIFY!

package {{.PackageName}}

import (
	"github.com/jmoiron/sqlx"
	"github.com/Peripli/service-manager/pkg/types"
	{{.StoragePackageImport}}
	{{.ApiPackageImport}}{{if .SupportsLabels}}
	"database/sql"
	"fmt"
	"time"
	"github.com/gofrs/uuid"{{end}}
)

func ({{.Type}}) NewInstance() {{.StoragePackage}}Entity {
	return {{.Type}}{}
}

func ({{.Type}}) PrimaryColumn() string {
	return "id"
}

func ({{.Type}}) TableName() string {
	return "{{.TableName}}"
}

func (e {{.Type}}) Labels() {{.StoragePackage}}EntityLabels {
	{{ if .SupportsLabels }} return {{.TypeLower}}Labels{} {{ else }} return nil {{ end }} 
}

func (e {{.Type}}) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	{{ if .SupportsLabels }}entities := make(map[string]*{{.ApiPackage}}{{.Type}})
	labels := make(map[string]map[string][]string)
	result := &{{.ApiPackage}}{{.Type}}s{
		{{.Type}}s: make([]*{{.ApiPackage}}{{.Type}}, 0),
	}
	for rows.Next() {
		row := struct {
			*{{.Type}}
			*{{.Type}}Label ` + "`db:\"{{.TypeLower}}_labels\"`" + `
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		entity, ok := entities[row.{{.Type}}.ID]
		if !ok {
			entity = row.{{.Type}}.ToObject().(*{{.ApiPackage}}{{.Type}})
			entities[row.{{.Type}}.ID] = entity
			result.{{.Type}}s = append(result.{{.Type}}s, entity)
		}
		if labels[entity.ID] == nil {
			labels[entity.ID] = make(map[string][]string)
		}
		labels[entity.ID][row.{{.Type}}Label.Key.String] = append(labels[entity.ID][row.{{.Type}}Label.Key.String], row.{{.Type}}Label.Val.String)
	}

	for _, b := range result.{{.Type}}s {
		b.Labels = labels[b.ID]
	}
	return result, nil {{ else }}result := &{{.ApiPackage}}{{.Type}}s{}
	for rows.Next() {
		var item {{.Type}}
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		result.Add(item.ToObject())
	}
	return result, nil{{ end }}
}
{{if .SupportsLabels}}
type {{.Type}}Label struct {
	*BaseLabel
	{{.Type}}ID  sql.NullString ` + "`db:\"{{.TypeLower}}_id\"`" + `
}

func (el {{.Type}}Label) TableName() string {
	return "{{.TypeLower}}_labels"
}

func (el {{.Type}}Label) PrimaryColumn() string {
	return "id"
}

func (el {{.Type}}Label) ReferenceColumn() string {
	return "{{.TypeLower}}_id"
}

func (el {{.Type}}Label) NewInstance() {{.StoragePackage}}Label {
	return {{.Type}}Label{}
}

func (el {{.Type}}Label) New(entityID, id, key, value string) {{.StoragePackage}}Label {
	now := time.Now()
	return {{.Type}}Label{
		BaseLabel: &BaseLabel{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: &now,
			UpdatedAt: &now,
		},
		{{.Type}}ID:  sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

type {{.TypeLower}}Labels []*{{.Type}}Label

func (el {{.TypeLower}}Labels) Single() {{.StoragePackage}}Label {
	return &{{.Type}}Label{}
}

func (el {{.TypeLower}}Labels) FromDTO(entityID string, labels types.Labels) ([]{{.StoragePackage}}Label, error) {
	var result []{{.StoragePackage}}Label
	now := time.Now()
	for key, values := range labels {
		for _, labelValue := range values {
			UUID, err := uuid.NewV4()
			if err != nil {
				return nil, fmt.Errorf("could not generate GUID for broker label: %s", err)
			}
			id := UUID.String()
			bLabel := &{{.Type}}Label{
				BaseLabel: &BaseLabel{
					ID:        sql.NullString{String: id, Valid: id != ""},
					Key:       sql.NullString{String: key, Valid: key != ""},
					Val:       sql.NullString{String: labelValue, Valid: labelValue != ""},
					CreatedAt: &now,
					UpdatedAt: &now,
				},
				{{.Type}}ID:  sql.NullString{String: entityID, Valid: entityID != ""},
			}
			result = append(result, bLabel)
		}
	}
	return result, nil
}

func (els {{.TypeLower}}Labels) ToDTO() types.Labels {
	labelValues := make(map[string][]string)
	for _, label := range els {
		values, exists := labelValues[label.Key.String]
		if exists {
			labelValues[label.Key.String] = append(values, label.Val.String)
		} else {
			labelValues[label.Key.String] = []string{label.Val.String}
		}
	}
	return labelValues
}
{{end}}
`
