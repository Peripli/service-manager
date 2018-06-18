package rest

import (
	"path"

	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/sirupsen/logrus"
)

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
		return true
	}
	// TODO: add support for **
	match, err := path.Match(pattern, endpointPath)
	if err != nil {
		logrus.Fatalf("Invalid endpoint path pattern %s: %v", endpointPath, err)
	}
	return match
}

func matchMethod(method string, methods []string) bool {
	if methods == nil {
		return true
	}
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}
