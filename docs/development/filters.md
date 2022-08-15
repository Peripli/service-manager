# Filters

Filters provide means to intercept any HTTP request to the Service Manager. It allows adding
custom logic before the request reaches the actual handler (HTTP Endpoint logic) and also
before it returns the response. Filters can either propagate a request to the next filter
in the chain or stop the request and write their own response.

Service Manager HTTP endpoints are described in the [API Specification](https://github.com/Peripli/specification/blob/master/api.md).

There are several types one has to know about. They can be found [here](pkg/web/types.go)

## Example: Request Logging Filter

```go
package myfilter

import "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"

// Filter implements web.Filter (a Named Middleware that matches (runs on)
// certain conditions called FilterMatchers)
type Filter struct{}

// Name implements web.Named
func (f *Filter) Name() string { return "FilterName" }

// Run implements web.Middleware
func (f *Filter) Run(r *web.Request, next web.Handler) (*web.Response, error) {
    // pre processing logic
    fmt.Printf("request with method %s to URL %s\n", r.Method, r.URL)

    // call next filter in chain
    resp, err := next.Handle(r)

    // add post processing logic
    fmt.Printf("response with status code %d and error %v\n", resp.StatusCode, err)

    return resp, err
}

// FilterMatchers that specify when the filter should run. Each
// FilterMatcher consists of Matchers that represent one endpoint ("place")
// where the filter should run.
// For example the following matches GET requests on /v1/platforms/** and
// all requests on /v1/service_brokers/**
func (f *Filter) FilterMatchers() []web.FilterMatcher {
    return []web.FilterMatcher{
        {
            // Matches all GET requests on /v1/platforms/**
            Matchers: []web.Matcher{
                web.Path("/v1/platforms/**"),
                web.Method("GET"),
            },
        },
        {
            // Matches all requests on /v1/service_brokers/**
            Matchers: []web.Matcher{
                web.Path("/v1/service_brokers/**"),
            },
        },
    }
}
```