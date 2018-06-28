package rest

import (
	"strings"

	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"
)

// MatchFilters returns all filters that mach the provided endpoint's path and method
func MatchFilters(endpoint *Endpoint, filters []filter.Filter) []filter.Filter {
	matches := []filter.Filter{}
	for _, filter := range filters {
		if matchPath(endpoint.Path, filter.PathPattern) &&
			matchMethod(endpoint.Method, filter.Methods) {
			matches = append(matches, filter)
		}
	}
	logrus.Debugf("%d filters for endpoint %v", len(matches), endpoint)
	return matches
}

func matchPath(endpointPath string, pattern string) bool {
	if pattern == "" {
		return false
	}
	pat, err := glob.Compile(pattern, '/')
	if err != nil {
		logrus.Panicf("Invalid endpoint path pattern %s: %v", pattern, err)
	}
	return pat.Match(endpointPath) || pat.Match(endpointPath+"/")
}

func matchMethod(method string, methods []string) bool {
	if methods == nil {
		return false
	}

	if len(methods) == 1 && methods[0] == "*" {
		return true
	}

	for _, m := range methods {
		if strings.ToLower(m) == strings.ToLower(method) {
			return true
		}
	}
	return false
}
