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
	"flag"
	"fmt"
	"os"
	"strings"
)

// generate <api/storage> <name> <labels/none>
func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		panic("Usage is <api/storage> <type_name> <labels/none> <api_package/none> <table_name/none>")
	}
	generationTarget := args[0]
	if generationTarget == "api" {

	}
	typeName := strings.Title(args[1])
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	lastIndexOfSlash := strings.LastIndex(dir, "/")
	packageName := dir
	if lastIndexOfSlash > 0 {
		packageName = dir[lastIndexOfSlash+1:]
	}
	fmt.Println(dir)
	fmt.Println(packageName)
	supportsLabels := len(args) > 2 && strings.ToLower(args[2]) == "labels"
	switch generationTarget {
	case "api":
		if err := GenerateApiTypeFile(dir, packageName, typeName, supportsLabels); err != nil {
			panic(err)
		}
	case "storage":
		var apiPackage, tableName string
		if len(args) > 3 {
			apiPackage = args[3]
		}
		if len(args) > 4 {
			tableName = args[4]
		}
		if err := GenerateStorageEntityFile(dir, typeName, packageName, apiPackage, tableName, supportsLabels); err != nil {
			panic(err)
		}
	default:
		panic(fmt.Sprintf("Unsupported generation type %s", generationTarget))
	}
}
