package filter

import (
	"encoding/json"
	"net/http"
)

type Request struct {
	*http.Request
	PathParams map[string]string
	Body       []byte
}

func (r *Request) String() string {
	return stringify(r)
}

type Response struct {
	// StatusCode is the HTTP status code
	StatusCode int

	Header http.Header
	Body   []byte
}

func (r *Response) String() string {
	return stringify(r)
}

func stringify(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

type Handler func(*Request) (*Response, error)
type Middleware func(req *Request, next Handler) (*Response, error)

type RequestMatcher struct {
	// Methods match request method
	// if nil, matches any method
	// NOTE: This will work as long as each route handles a single method.
	// If a route handles multiple methods (e.g. *),
	// the filter could be called for methods which are not listed here.
	Methods []string

	// PathPattern matches endpoint path (as registered in mux)
	// This is a glob pattern so you can use * and **.
	// If empty, matches any path.
	PathPattern string
}

type Filter struct {
	RequestMatcher
	Middleware
}
