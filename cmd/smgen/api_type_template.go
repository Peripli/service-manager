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

const ApiTypeTemplate = `// GENERATED. DO NOT MODIFY!

package {{.PackageName}}

import (
	"encoding/json"
{{if .TypesPackageImport}}
	{{.TypesPackageImport}}
{{end}}
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const {{.Type}}Type {{.TypesPackage}}ObjectType = web.{{.TypePlural}}URL

type {{.TypePlural}} struct {
	{{.TypePlural}} []*{{.Type}} ` + "`json:\"{{.TypePluralLowercase}}\"`" + `
}

func (e *{{.TypePlural}}) Add(object {{.TypesPackage}}Object) {
	e.{{.TypePlural}} = append(e.{{.TypePlural}}, object.(*{{.Type}}))
}

func (e *{{.TypePlural}}) ItemAt(index int) {{.TypesPackage}}Object {
	return e.{{.TypePlural}}[index]
}

func (e *{{.TypePlural}}) Len() int {
	return len(e.{{.TypePlural}})
}

func (e *{{.Type}}) GetType() {{.TypesPackage}}ObjectType {
	return {{.Type}}Type
}

// MarshalJSON override json serialization for http response
func (e *{{.Type}}) MarshalJSON() ([]byte, error) {
	type E {{.Type}}
	toMarshal := struct {
		*E
		CreatedAt *string ` + "`json:\"created_at,omitempty\"`" + `
		UpdatedAt *string ` + "`json:\"updated_at,omitempty\"`" + `
		Labels    Labels  ` + "`json:\"labels,omitempty\"`" + `
	}{
		E:      (*E)(e),
		Labels: e.Labels,
	}
	if !e.CreatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	hasNoLabels := true
	for key, values := range e.Labels {
		if key != "" && len(values) != 0 {
			hasNoLabels = false
			break
		}
	}
	if hasNoLabels {
		toMarshal.Labels = nil
	}
	return json.Marshal(toMarshal)
}
`
