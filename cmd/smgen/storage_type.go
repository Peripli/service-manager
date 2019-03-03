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
)

const StorageTypesDirectory = "github.com/Peripli/service-manager/storage/postgres"

type StorageType struct {
	Type                 string
	TypeLower            string
	TableName            string
	PackageName          string
	SupportsLabels       bool
	ApiPackage           string
	ApiPackageImport     string
	StoragePackage       string
	StoragePackageImport string
}

func GenerateStorageEntityFile(storageTypeDir, typeName, packageName, apiPackageDir, tableName string, supportsLabels bool) error {
	if tableName == "" {
		tableName = strings.ToLower(typeName) + "s"
	}
	apiPackage := "types."
	apiPackageImport := ""
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
		storagePackageImport = StorageTypesDirectory
	}
	t := template.Must(template.New("generate-storage-type").Parse(STORAGE_TYPE_TEMPLATE))
	entityTemplate := StorageType{
		Type:                 typeName,
		TypeLower:            strings.ToLower(typeName),
		TableName:            tableName,
		SupportsLabels:       supportsLabels,
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
