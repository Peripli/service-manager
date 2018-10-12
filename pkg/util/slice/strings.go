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

type stringPredicate func(next string) bool

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
	return filter(stringSlice, func(next string) bool {
		return strings.Contains(next, str)
	})
}

// StringsAnyPrefix returns true if any of the strings in the slice have the given prefix.
func StringsAnyPrefix(stringSlice []string, prefix string) bool {
	stringsWithPrefix := filter(stringSlice, func(next string) bool {
		return strings.HasPrefix(next, prefix)
	})
	return len(stringsWithPrefix) > 0
}

// StringsAnyEquals returns true if any of the strings in the slice equal the given string.
func StringsAnyEquals(stringSlice []string, str string) bool {
	equalStrings := filter(stringSlice, func(next string) bool {
		return next == str
	})
	return len(equalStrings) > 0
}

// StringsAnyEqualsIgnoreCase returns true if any of the strings in the slice equal the given string ignoring the case.
func StringsAnyEqualsIgnoreCase(stringSlice []string, str string) bool {
	equalIgnoreCase := filter(stringSlice, func(next string) bool {
		return strings.EqualFold(next, str)
	})
	return len(equalIgnoreCase) > 0
}

func filter(stringSlice []string, predicate stringPredicate) []string {
	var result []string
	for _, v := range stringSlice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}
