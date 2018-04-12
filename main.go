/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package main

import (
	"fmt"
	"github.com/fatih/structs"
	"strings"
)

type Settings struct {
	Server serverSettings
	Db     dbSettings
	Log    logSettings
	Mocked mockSettings
}

type nestedSettings struct {
	Hello string
}

type mockSettings struct {
	Nested nestedSettings
	Bye    string
}

type serverSettings struct {
	Port            int
	RequestTimeout  int
	ShutdownTimeout int
}

type dbSettings struct {
	URI string
}

type logSettings struct {
	Level  string
	Format string
}

func main() {
	fmt.Println(bindStruct(&Settings{}))
}
func bindStruct(value interface{}) []string {
	var result []string

	pathBuilder(value, "", &result)
	return result
}

func pathBuilder(value interface{}, buffer string, result *[]string) {
	if !structs.IsStruct(value) {
		index := strings.LastIndex(buffer, ".")
		if index == -1 {
			index = 0
		}
		*result = append(*result, strings.ToLower(buffer[0:index]))
		return
	}
	s := structs.New(value)
	for _, field := range s.Fields() {
		if field.IsExported() {
			if !field.IsEmbedded() {
				buffer += field.Name() + "."
			}
			pathBuilder(field.Value(), buffer, result)
			if !field.IsEmbedded() {
				buffer = buffer[0:strings.LastIndex(buffer, field.Name())]
			}
		}
	}
}
