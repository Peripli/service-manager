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

package env

import (
	"reflect"
	"strings"

	"github.com/fatih/structs"
)

type descriptionTree struct {
	Value    string
	Children []*descriptionTree
}

func newDescriptionTree(root string) *descriptionTree {
	return &descriptionTree{
		Value:    root,
		Children: nil,
	}
}

func (t *descriptionTree) AddNode(tree *descriptionTree) {
	if t.Children == nil {
		t.Children = []*descriptionTree{tree}
		return
	}
	t.Children = append(t.Children, tree)
}

type configurationParameter struct {
	Name         string
	DefaultValue interface{}
}

func buildParametersAndDescriptions(value interface{}) ([]configurationParameter, []string) {
	tree := &descriptionTree{}
	var parameters []configurationParameter
	buildDescriptionTreeWithParameters(value, tree, "", &parameters)
	return parameters, buildDescriptionsFromTree(tree)
}

func buildParameters(value interface{}) []configurationParameter {
	tree := &descriptionTree{}
	var parameters []configurationParameter
	buildDescriptionTreeWithParameters(value, tree, "", &parameters)
	return parameters
}

func buildDescriptionsFromTree(tree *descriptionTree) []string {
	return buildDescriptionPaths(tree, []*descriptionTree{})
}

func buildDescriptionPaths(root *descriptionTree, path []*descriptionTree) []string {
	path = append(path, root)
	var result []string
	if root.Children == nil || len(root.Children) == 0 {
		res := ""
		for _, node := range path {
			res += node.Value
		}
		if res == "" {
			res = "external configuration or no description provided"
		}
		if res != "" {
			result = append(result, res)
		}
		return result
	}
	for _, node := range root.Children {
		result = append(result, buildDescriptionPaths(node, path)...)
	}
	return result
}

func buildDescriptionTreeWithParameters(value interface{}, tree *descriptionTree, buffer string, result *[]configurationParameter) {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map:
		for _, key := range v.MapKeys() {
			field := v.MapIndex(key).Interface()
			if isValidField(field) {
				name := key.String()
				buffer += name + "."
				buildDescriptionTreeWithParameters(field, tree, buffer, result)
				buffer = buffer[0:strings.LastIndex(buffer, name)]
			}
		}
	default:
		if !structs.IsStruct(value) {
			index := strings.LastIndex(buffer, ".")
			if index == -1 {
				index = 0
			}
			key := strings.ToLower(buffer[0:index])
			*result = append(*result, configurationParameter{Name: key, DefaultValue: value})
			tree.Children = nil
			return
		}
		s := structs.New(value)
		for _, field := range s.Fields() {
			if isValidField(field) {
				var name string
				if field.Tag("mapstructure") != "" {
					name = field.Tag("mapstructure")
				} else {
					name = field.Name()
				}
				if name == "-" {
					continue
				}
				if name == ",squash" {
					buildDescriptionTreeWithParameters(field.Value(), tree, buffer, result)
					continue
				}
				description := ""
				if field.Tag("description") != "" {
					description = field.Tag("description")
				}

				baseTree := newDescriptionTree(description)
				tree.AddNode(baseTree)
				buildDescriptionTreeWithParameters(field.Value(), baseTree, buffer+name+".", result)
			}
		}
	}
}

func isValidField(field interface{}) bool {
	if fieldStruct, ok := field.(*structs.Field); ok {
		kind := fieldStruct.Kind()
		return fieldStruct.IsExported() && kind != reflect.Interface && kind != reflect.Func
	}
	kind := reflect.ValueOf(field).Kind()
	return kind != reflect.Interface && kind != reflect.Func
}
