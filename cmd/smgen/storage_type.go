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

const StorageTypesDirectory = "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/postgres"

type StorageType struct {
	Type                 string
	TypeLower            string
	TypeLowerSnakeCase   string
	TypePlural           string
	TableName            string
	ApiType              string
	ApiTypePlural        string
	PackageName          string
	ApiPackage           string
	ApiPackageImport     string
	StoragePackage       string
	StoragePackageImport string
}

func GenerateStorageEntityFile(storageTypeDir, typeName, packageName, apiPackageDir, tableName string) error {
	typeNamePlural := toPlural(typeName)
	if tableName == "" {
		tableName = toLowerSnakeCase(typeNamePlural)
	}
	apiPackage := "types."
	apiPackageImport := ""
	apiType := typeName
	apiTypePlural := typeNamePlural
	if strings.ContainsRune(apiPackageDir, ':') {
		parts := strings.Split(apiPackageDir, ":")
		apiPackageDir = parts[0]
		apiType = parts[1]
		apiTypePlural = toPlural(apiType)
	}
	if !strings.Contains(apiPackageDir, APITypesDirectory) {
		lastIndexOfSlash := strings.LastIndex(apiPackageDir, "/")
		if lastIndexOfSlash > 0 {
			apiPackage = apiPackageDir[lastIndexOfSlash+1:]
		}
		apiPackageImport = fmt.Sprintf(`"%s"`, apiPackageDir)
	}
	storagePackageImport := ""
	storagePackage := ""

	if !strings.Contains(storageTypeDir, StorageTypesDirectory) {
		storagePackage = "postgres."
		storagePackageImport = fmt.Sprintf("\"%s\"", StorageTypesDirectory)
	}

	t := template.Must(template.New("generate-storage-type").Parse(StorageTypeTemplate))
	entityTemplate := StorageType{
		Type:                 typeName,
		TypeLowerSnakeCase:   toLowerSnakeCase(typeName),
		TypePlural:           typeNamePlural,
		TableName:            tableName,
		ApiType:              apiType,
		ApiTypePlural:        apiTypePlural,
		PackageName:          packageName,
		ApiPackage:           apiPackage,
		ApiPackageImport:     apiPackageImport,
		StoragePackage:       storagePackage,
		StoragePackageImport: storagePackageImport,
	}
	file, err := os.Create(fmt.Sprintf("%s/%s_gen.go", storageTypeDir, strings.ToLower(typeName)))
	if err != nil {
		return err
	}
	if err = t.Execute(file, entityTemplate); err != nil {
		return err
	}
	return nil
}

func toPlural(typeName string) string {
	typeNamePlural := fmt.Sprintf("%ss", typeName)
	if strings.HasSuffix(typeName, "y") {
		typeNamePlural = fmt.Sprintf("%sies", typeName[:len(typeName)-1])
	}
	return typeNamePlural
}

func toLowerSnakeCase(str string) string {
	builder := strings.Builder{}
	for i, char := range str {
		if unicode.IsUpper(char) && i > 0 {
			builder.WriteRune('_')
		}
		builder.WriteRune(unicode.ToLower(char))
	}
	return builder.String()
}
