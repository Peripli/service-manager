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

const StorageTypeTemplate = `// GENERATED. DO NOT MODIFY!

package {{.PackageName}}

import (
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
{{if .StoragePackageImport}}
	{{.StoragePackageImport}}
{{end}}{{if .ApiPackageImport}}
	{{.ApiPackageImport}}
{{end}}
	"database/sql"
	"time"
)

var _ {{.StoragePackage}}PostgresEntity = &{{.Type}}{}

const {{.Type}}Table = "{{.TableName}}"

func (*{{.Type}}) LabelEntity() {{.StoragePackage}}PostgresLabel {
	return &{{.Type}}Label{}
}

func (*{{.Type}}) TableName() string {
	return {{.Type}}Table
}

func (e *{{.Type}}) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &{{.Type}}Label{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{{.Type}}ID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *{{.Type}}) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*{{.Type}}
			{{.Type}}Label ` + "`db:\"{{.TypeLowerSnakeCase}}_labels\"`" + `
		}{}
	}
	result := &{{.ApiPackage}}{{.ApiTypePlural}}{
		{{.ApiTypePlural}}: make([]*{{.ApiPackage}}{{.ApiType}}, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type {{.Type}}Label struct {
	BaseLabelEntity
	{{.Type}}ID sql.NullString ` + "`db:\"{{.TypeLowerSnakeCase}}_id\"`" + `
}

func (el {{.Type}}Label) LabelsTableName() string {
	return "{{.TypeLowerSnakeCase}}_labels"
}

func (el {{.Type}}Label) ReferenceColumn() string {
	return "{{.TypeLowerSnakeCase}}_id"
}
`
