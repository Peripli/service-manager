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
	"net/http"

	"fmt"

	"github.com/sirupsen/logrus"
)

// Request contains the original http.Request, path parameters and the raw body
// Request.Request.Body should not be used as it would be already processed by internal implementation
type Request struct {
	// Request is the original http.Request
	*http.Request

	// PathParams contains the URL path parameters
	PathParams map[string]string

	// Body is the loaded request body (usually JSON)
	Body []byte
}

// Response defines the attributes of the HTTP response that will be sent to the client
type Response struct {
	// StatusCode is the HTTP status code
	StatusCode int

	// Header contains the response headers
	Header http.Header

	// Body is the response body (usually JSON)
	Body []byte
}

type Named interface {
	Name() string
}

//go:generate counterfeiter . Handler
type Handler interface {
	Handle(req *Request) (resp *Response, err error)
}

type HandlerFunc func(req *Request) (resp *Response, err error)

func (rhf HandlerFunc) Handle(req *Request) (resp *Response, err error) {
	return rhf(req)
}

type Middleware interface {
	Run(next Handler) Handler
}

type MiddlewareFunc func(handler Handler) Handler

func (mf MiddlewareFunc) Run(handler Handler) Handler {
	return mf(handler)
}

type Matcher interface {
	Matches(route Route) (bool, error)
}

type MatcherFunc func(route Route) (bool, error)

func (m MatcherFunc) Matches(route Route) (bool, error) {
	return m(route)
}

type RouteMatcher struct {
	Matchers []Matcher
}

type Filter interface {
	Named
	Middleware

	RouteMatchers() []RouteMatcher
}

type Middlewares []Middleware

func (ms Middlewares) Chain(h Handler) Handler {
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i].Run(h)
	}
	return h
}

type Filters []Filter

// ChainMatching builds a pkg/web.Handler that chains up the filters that match the provided route and the actual handler
func (fs Filters) ChainMatching(route Route) Handler {
	filters := fs.Matching(route)
	return filters.Chain(route.Handler)
}

func (fs Filters) Chain(h Handler) Handler {
	wrappedFilters := make([]Handler, len(fs)+1)
	wrappedFilters[len(fs)] = h

	for i := len(fs) - 1; i >= 0; i-- {
		i := i
		wrappedFilters[i] = HandlerFunc(func(request *Request) (*Response, error) {
			logrus.Debug("Entering Filter: ", fs[i].Name(), " Path: ", request.URL.Path, " Method: ", request.Method)
			resp, err := fs[i].Run(wrappedFilters[i+1]).Handle(request)
			logrus.Debug("Exiting Filter: ", fs[i].Name(), " Err: ", err, " Path: ", request.URL.Path, " Method: ", request.Method)
			return resp, err
		})
	}

	return wrappedFilters[0]
}

func (fs Filters) Matching(route Route) Filters {
	matchedFilters := make([]Filter, 0)
	matchedNames := make([]string, 0)
	for _, filter := range fs {
		for _, routeMatcher := range filter.RouteMatchers() {
			missMatch := false
			for _, matcher := range routeMatcher.Matchers {
				match, err := matcher.Matches(route)
				if err != nil {
					panic(fmt.Sprintf("error matching filter %s: %s", filter.Name(), err.Error()))
				}
				if !match {
					missMatch = true
					break
				}
			}
			if !missMatch {
				matchedFilters = append(matchedFilters, filter)
				matchedNames = append(matchedNames, filter.Name())
				break
			}
		}
	}
	logrus.Debugf("Filters for endpoint %s:%s: [%v]", route.Endpoint.Path, route.Endpoint.Method, matchedNames)
	return matchedFilters
}
