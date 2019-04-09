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
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
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

// Named is an interface that objects that need to be identified by a particular name should implement.
type Named interface {
	// Name returns the string identifier for the object
	Name() string
}

// Handler is an interface that objects can implement to be registered in the SM REST API.
//go:generate counterfeiter . Handler
type Handler interface {
	// Handle processes a Request and returns a corresponding Response or error
	Handle(req *Request) (resp *Response, err error)
}

// HandlerFunc is an adapter that allows to use regular functions as Handler interface implementations.
type HandlerFunc func(req *Request) (resp *Response, err error)

// Handle allows HandlerFunc to act as a Handler
func (rhf HandlerFunc) Handle(req *Request) (resp *Response, err error) {
	return rhf(req)
}

// Middleware is an interface that objects that should act as filters or plugins need to implement. It intercepts
// the request before reaching the final handler and allows during preprocessing and postprocessing.
type Middleware interface {
	// Run returns a handler that contains the handling logic of the Middleware. The implementation of Run
	// should invoke next's Handle if the request should be chained to the next Handler.
	// It may also terminate the request by not invoking the next Handler.
	Run(req *Request, next Handler) (*Response, error)
}

// MiddlewareFunc is an adapter that allows to use regular functions as Middleware
type MiddlewareFunc func(req *Request, next Handler) (*Response, error)

// Run allows MiddlewareFunc to act as a Middleware
func (mf MiddlewareFunc) Run(req *Request, handler Handler) (*Response, error) {
	return mf(req, handler)
}

// Matcher allows checking whether an Endpoint matches a particular condition
type Matcher interface {
	// Matches matches a route against a particular condition
	Matches(endpoint Endpoint) (bool, error)
}

// MatcherFunc is an adapter that allows regular functions to act as Matchers
type MatcherFunc func(endpoint Endpoint) (bool, error)

// Matches allows MatcherFunc to act as a Matcher
func (m MatcherFunc) Matches(endpoint Endpoint) (bool, error) {
	return m(endpoint)
}

// FilterMatcher type represents a set of conditions (Matchers) that need to match if order
// for a FilterMatcher to match
type FilterMatcher struct {
	// Matchers represents a set of conditions that need to be matched
	Matchers []Matcher
}

// Filter is an interface that Named Middlewares that should satisfy a set of Matchers (conditions) should implement.
//go:generate counterfeiter . Filter
type Filter interface {
	Named
	Middleware

	// FilterMatchers Returns a set of FilterMatchers each containing a set of Matchers. Each FilterMatcher represents
	// one place where the Filter would run.
	FilterMatchers() []FilterMatcher
}

// Filters represents a slice of Filter elements
type Filters []Filter

// ChainMatching builds a pkg/web.Handler that chains up the filters that match the provided route and
// the actual handler
func (fs Filters) ChainMatching(route Route) Handler {
	filters := fs.Matching(route.Endpoint)
	return filters.Chain(route.Handler)
}

// Chain chains the Filters around the specified Handler and returns a Handler. It also adds logic for logging before
// entering and after exiting from filters.
func (fs Filters) Chain(h Handler) Handler {
	wrappedFilters := make([]Handler, len(fs)+1)
	wrappedFilters[len(fs)] = h

	for i := len(fs) - 1; i >= 0; i-- {
		i := i
		wrappedFilters[i] = HandlerFunc(func(r *Request) (*Response, error) {
			params := map[string]interface{}{
				"path":                 r.URL.Path,
				"method":               r.Method,
				log.FieldCorrelationID: log.CorrelationIDForRequest(r.Request),
			}
			logger := log.C(r.Context())
			logger.WithFields(params).Debug("Entering Filter: ", fs[i].Name())

			resp, err := fs[i].Run(r, wrappedFilters[i+1])

			params["err"] = err
			if resp != nil {
				params["statusCode"] = resp.StatusCode
			}

			logger.WithFields(params).Debug("Exiting Filter: ", fs[i].Name())

			return resp, err
		})
	}

	return wrappedFilters[0]
}

// Matching returns a subset of Filters that match the specified endpoint
func (fs Filters) Matching(endpoint Endpoint) Filters {
	matchedFilters := make([]Filter, 0)
	matchedNames := make([]string, 0)
	for _, filter := range fs {
		if len(filter.FilterMatchers()) == 0 {
			matchedFilters = append(matchedFilters, filter)
			matchedNames = append(matchedNames, filter.Name())
		}
		for _, routeMatcher := range filter.FilterMatchers() {
			missMatch := false
			for _, matcher := range routeMatcher.Matchers {
				match, err := matcher.Matches(endpoint)
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
	log.D().Debugf("Filters for %s %s:%v", endpoint.Method, endpoint.Path, matchedNames)
	return matchedFilters
}
