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

package slice

import "strings"

// StringsIntersection returns the common elements in two string slices.
func StringsIntersection(str1, str2 []string) []string {
	var intersection []string
	strings := make(map[string]bool)
	for _, v := range str1 {
		strings[v] = true
	}
	for _, v := range str2 {
		if strings[v] {
			intersection = append(intersection, v)
		}
	}
	return intersection
}

// StringsContaining returns a slice of the strings containing the specified string.
func StringsContaining(stringSlice []string, str string) []string {
	var result []string
	for _, v := range stringSlice {
		if strings.Contains(v, str) {
			result = append(result, v)
		}
	}
	return result
}

// StringsAnyPrefix returns true if any of the strings in the slice have the given prefix.
func StringsAnyPrefix(stringSlice []string, prefix string) bool {
	for _, v := range stringSlice {
		if strings.HasPrefix(v, prefix) {
			return true
		}
	}
	return false
}

// StringsAnyEquals returns true if any of the strings in the slice equal the given string.
func StringsAnyEquals(stringSlice []string, str string) bool {
	for _, v := range stringSlice {
		if v == str {
			return true
		}
	}
	return false
}
