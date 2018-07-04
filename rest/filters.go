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
