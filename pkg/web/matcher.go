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

package web

import (
	"errors"

	"github.com/gobwas/glob"
)

var (
	errEmptyPathPattern   = errors.New("empty path pattern not allowed")
	errInvalidPathPattern = errors.New("invalid path pattern")
	errEmptyHTTPMethods   = errors.New("empty http methods not allowed")
)

// Methods returns a Matcher that matches to routes that contain any of the specified methods
func Methods(methods ...string) Matcher {
	return MatcherFunc(func(endpoint Endpoint) (bool, error) {
		if len(methods) == 0 {
			return false, errEmptyHTTPMethods
		}
		method := endpoint.Method
		return matchInArray(methods, method), nil
	})
}

// Path returns a Matcher that matches to routes a path that matches any of the specified patterns
func Path(patterns ...string) Matcher {
	return MatcherFunc(func(endpoint Endpoint) (bool, error) {
		path := endpoint.Path
		if len(patterns) == 0 {
			return false, errEmptyPathPattern
		}

		for _, pattern := range patterns {
			pat, err := glob.Compile(pattern, '/')
			if err != nil {
				return false, errInvalidPathPattern
			}
			if pat.Match(path) || pat.Match(path+"/") {
				return true, nil
			}
		}
		return false, nil
	})

}

func matchInArray(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}
