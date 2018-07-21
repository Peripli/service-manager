package web

import (
	"errors"

	"github.com/gobwas/glob"
)

var (
	errEmptyPathPattern   = errors.New("empty path pattern not allowed")
	errInvalidPathPattern = errors.New("invalid path pattern")
	errEmptyHttpMethods   = errors.New("empty http methods not allowed")
)

func Methods(methods ...string) Matcher {
	return MatcherFunc(func(route Route) (bool, error) {
		if len(methods) == 0 {
			return false, errEmptyHttpMethods
		}
		method := route.Endpoint.Method
		return matchInArray(methods, method), nil
	})
}

func Path(patterns ...string) Matcher {
	return MatcherFunc(func(route Route) (bool, error) {
		path := route.Endpoint.Path
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
