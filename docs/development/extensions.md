# Extending the Service Manager

- [Filters](#filters)
- [Plugins](#plugins)
- [Controllers](#controllers)
- [Health](#provide-your-own-health-indicators)

The main extension points of the service manager are filters, plugins and controllers. The
interfaces that need to be implemented in order to provide an extension point can be found
in the `pkg/web` package.

In addition, for your components you can provide your own health metrics by adding a simple health indicator.
You can also configure how all the provided health metrics be displayed by registering your aggregation policy
which can customize the health report.

## Filters

Filters provide means to intercept any HTTP request to the Service Manager. It allows adding
custom logic before the request reaches the actual handler (HTTP Endpoint logic) and also
before it returns the response. Filters can either propagate a request to the next filter
in the chain or stop the request and write their own response.

Service Manager HTTP endpoints are described in the [API Specification](https://github.com/Peripli/specification/blob/master/api.md).

There are several types one has to know about. They can be found [here](pkg/web/types.go)

### Example: Request Logging Filter

```go
package myfilter

import "github.com/Peripli/service-manager/pkg/web"

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

## Plugins

Plugins provide means to intercept different OSB calls (provision, deprovision, bind, etc.).
It can modify both the request before it reaches the broker and the response before being sent to the client.

There are several interfaces that the plugin can implement for different OSB API operations.
They can be found `pkg/web/plugin.go`.
For each OSB operation intercepted by the plugin, Service Manager creates a new filter for the respective HTTP endpoint.

### Example: Catalog modification plugin

For example a plugin that modifies the catalog can be written as follows:

```go
package myplugin

import (
    "github.com/tidwall/sjson"
    "github.com/Peripli/service-manager/pkg/web"
)

type MyPlugin struct {}

func (p *MyPlugin) Name() string { return "MyPlugin" }

func (p *MyPlugin) FetchCatalog(req *web.Request, next web.Handler) (*web.Response, error) {
    res, err := next.Handle(req)
    if err != nil {
        return nil, err
    }
    serviceName := gjson.GetBytes(res.Body, "services.0.name").String()
    res.Body, err = sjson.SetBytes(res.Body, "services.0.name", serviceName+"-suffix")
    return res, err
}
```

### Example: Response code modification plugin

This plugin will implement another interface on the plugin from the previous section.
It will modify the response status code for provision operation.

```go
func (p *MyPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
    if !checkCredentials(req) {
        return &web.Response{
            StatusCode: 401 // Unauthorized
        }, nil
    }
    return next.Handle(req)
}
```

## Best practices for writing filters and plugins

### Request and response body modifications

Request and response work with plain byte arrays (usually JSON). That's why it is recommended:

- For JSON modification (as in the [catalog plugin](#catalog-modification-plugin)) use [sjson](https://github.com/tidwall/sjson)

- To extract some value from JSON use [gjson](https://github.com/tidwall/gjson)

- **NOTE:** Be aware that JSON request and response may contain non-standard properties.

So when modifying the JSON body make sure to preserve them.
For example avoid marshalling from fixed structures.

### Do not modify the original Request object

`web.Request` contains the original `http.Request` object, but it **should NOT** be modified as this might lead to undesired behaviors.

**Do NOT** use the `http.Request.Body` as this reader is already processed by Service Manager code. The body can be accessed from `web.Request.Body` which is a byte array.

### Chaining

As part of execution of filter/plugin code call the `next.Handle(req)` function, this will forward control to the next filter/plugin in the chain.
In case filter/plugin logic requires to stop the chain and exit, just omit the call to `next.Handle(req)` function.

### Error handling

In case of error, return `nil` response and an error object.
Use `util.HTTPError` function to send error information and status code to the HTTP client.
Use the `pkg/util` package for different utility methods for processing and creating requests, responses and errors.
All other errors will result in status 500 (Internal Server Error) being returned to the client.

## Controllers

Controllers provide means to add additional APIs to the Service Manager. The Service Manager `pkg/web` package exposes interfaces one should implement in order to add additional SM APIs.

### Example Controller

```go
...
// Controller
type MyController struct {
}

var _ web.Controller = &MyController{}

func (c *MyController) ping(r *web.Request) (*web.Response, error) {
    return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// Routes specifies the routes in which the controller should run and the handler that should be executed
func (c *MyController) Routes() []web.Route {
    return []web.Route{
        {
            Endpoint: web.Endpoint{
                Method: http.MethodGet,
                Path:   "/api/v1/monitor/health",
            },
            Handler: c.ping,
        },
    }
}
...
```

## Registering Extensions

In order to add this plugin to the Service Manager one has to do the following:

```go
...
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    env := sm.DefaultEnv()
    serviceManager := sm.New(ctx, cancel, env)

    serviceManager.RegisterPlugins(&myplugin.MyPlugin{})
    serviceManager.RegisterFilters(&myfilter.MyFilter{})
    serviceManager.RegisterController(&mycontroller.MyController{})

    sm := serviceManager.Build()
    sm.Run()
}
...
```

## Provide your own health indicators

You can add your own health metrics to be available on the health endpoint (`/v1/monitor/health`).
The calculated healths can then be formatted to your liking by registering an aggregation policy.

```go
...
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    env := sm.DefaultEnv()
    serviceManager := sm.New(ctx, cancel, env)
    serviceManager.AddHealthIndicator(&MyHealthIndicator{})
    serviceManager.RegisterHealthAggregationPolicy(&MyAggregationPolicy{})
    sm := serviceManager.Build()
    sm.Run()
}
```
