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

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"unicode"
)

const APITypesDirectory = "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"

type ApiType struct {
	PackageName         string
	TypePlural          string
	TypePluralLowercase string
	Type                string
	TypesPackageImport  string
	TypesPackage        string
}

func GenerateApiTypeFile(apiTypeDir, packageName, typeName string) error {
	typeNamePlural := fmt.Sprintf("%ss", typeName)
	if strings.HasSuffix(typeName, "y") {
		typeNamePlural = fmt.Sprintf("%sies", typeName[:len(typeName)-1])
	}
	t := template.Must(template.New("generate-api-type").Parse(ApiTypeTemplate))
	var typesPackageImport string
	typesPackage := ""
	if !strings.Contains(apiTypeDir, APITypesDirectory) {
		typesPackage = "types."
		typesPackageImport = fmt.Sprintf(`"%s"`, APITypesDirectory)
	}
	builder := strings.Builder{}
	for i, char := range typeNamePlural {
		if unicode.IsUpper(char) && i > 0 {
			builder.WriteRune('_')
		}
		builder.WriteRune(unicode.ToLower(char))
	}
	apiType := ApiType{
		PackageName:         packageName,
		TypePlural:          typeNamePlural,
		TypePluralLowercase: builder.String(),
		Type:                typeName,
		TypesPackage:        typesPackage,
		TypesPackageImport:  typesPackageImport,
	}
	file, err := os.Create(fmt.Sprintf("%s/%s_gen.go", apiTypeDir, strings.ToLower(typeName)))
	if err != nil {
		return err
	}
	if err = t.Execute(file, apiType); err != nil {
		return err
	}
	return nil
}
