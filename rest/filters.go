/*
 *    Copyright 2018 The Service Manager Authors
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

package rest

import (
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"
)

// MatchFilters returns all filters that mach the provided endpoint's path and method
func MatchFilters(endpoint *Endpoint, filters []web.Filter) []web.Filter {
	matches := []web.Filter{}
	for _, filter := range filters {
		if matchPath(endpoint.Path, filter.PathPattern, filter.Name) &&
			matchMethod(endpoint.Method, filter.Methods) {
			matches = append(matches, filter)
		}
	}
	if logrus.GetLevel() >= logrus.DebugLevel {
		logrus.Debugf("Filters for endpoint %v: [%v]", endpoint, filterNames(matches))
	}
	return matches
}

func filterNames(filters []web.Filter) string {
	names := make([]string, len(filters))
	for i, f := range filters {
		if f.Name == "" {
			names[i] = "<anonymous>"
		} else {
			names[i] = f.Name
		}
	}
	return strings.Join(names, ", ")
}

func matchPath(endpointPath, pattern, filterName string) bool {
	if pattern == "" {
		logrus.Panicf("Empty path pattern for filter %s", filterName)
	}
	pat, err := glob.Compile(pattern, '/')
	if err != nil {
		logrus.Panicf("Invalid endpoint path pattern %s: %v", pattern, err)
	}
	return pat.Match(endpointPath) || pat.Match(endpointPath+"/")
}

func matchMethod(method string, methods []string) bool {
	if methods == nil {
		return true
	}

	for _, m := range methods {
		if strings.ToLower(m) == strings.ToLower(method) {
			return true
		}
	}
	return false
}
