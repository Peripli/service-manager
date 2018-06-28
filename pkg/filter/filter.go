package filter

import (
	"net/http"
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

// Handler is a function that handles a request after all the applicable filters
type Handler func(*Request) (*Response, error)

// Middleware is a function that intercepts a request before it reaches the final handler.
// Normally this function should invoke next with the request to proceed with
// the next middleware or the final handler.
// It can also terminate the request processing by not invoking next.
type Middleware func(*Request, Handler) (*Response, error)

// RouteMatcher defines the criteria to match against registered routes.
//
// NOTE: This does not match against HTTP requests at runtime but registered
// routes at startup. This improves performance but has some limitations.
type RouteMatcher struct {
	// Methods match route method.
	// If it contains one element of value "*", matches any method.
	//
	// NOTE: This will work as long as each route handles a single method.
	// If a route handles multiple methods,
	// the filter could be called for methods which are not listed here.
	Methods []string

	// PathPattern matches route path (as registered in mux).
	// This is a glob pattern so you can use * and **.
	// ** matches also / but * does not.
	// If empty, will not match at all.
	PathPattern string
}

// Filter defines a function that can intercept requests on specific routes
type Filter struct {
	// RouteMatcher defines the criteria for which routes to apply this filter
	RouteMatcher

	// Middleware is a function that intercepts a request before it reaches the final handler.
	Middleware
}
