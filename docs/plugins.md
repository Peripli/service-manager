# Plugins and Filters

## Filters
Filters provide means to intercept any HTTP request to Service Manager.
Service Manager HTTP endpoints are described in the [API Specification](https://github.com/Peripli/specification/blob/master/api.md).

There are several types one has to know about. They can be found [here](pkg/web/types.go)

### Example: Request logging filter

```go
...

func requestLogging(req *web.Request, next web.Handler) (*web.Response, error) {
    res, err := next(req)
    if err != nil {
        fmt.Printf("%s request to URL %s completed with error %s",
            req.Method, req.URL, err)
    } else {
        fmt.Printf("%s request to URL %s completed with status %d",
            req.Method, req.URL, res.StatusCode)
    }
    return res, err
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    sm := servicemanager.New(ctx, cancel)
    sm.RegisterFilter(&web.Filter{
        RouterMatcher: &web.RouterMatcher{
            // Will match any HTTP request
            PathPattern: "**",
        },
        Middleware: requestLogging,
    })
    sm.Run()
}
...
```

## Plugins
Plugins provide means to intercept different OSB calls (provision, deprovision, bind, etc.).
It can modify both the request before it reaches the broker and the response before being sent to the client.

There are several interfaces that the plugin can implement for different OSB API operations.
They can be found [here](pkg/plugin/plugin.go).
For each OSB operation intercepted by the plugin, Service Manager creates a new filter for the respective HTTP endpoint.

There are several types one has to know about. They can be found [here](pkg/web/types.go)

### Example: Catalog modification plugin

For example a plugin that modifies the catalog can be written as follows:

```go
package mypackage

import (
    "github.com/tidwall/sjson"
)

type MyPlugin struct {}

func (p *MyPlugin) Name() string { return "MyPlugin" }

func (p *MyPlugin) FetchCatalog(req *web.Request, next web.Handler) (*web.Response, error) {
    res, err := next(req)
    if err != nil {
        return nil, err
    }
    serviceName := gjson.GetBytes(res.Body, "services.0.name").String()
    res.Body, err = sjson.SetBytes(res.Body, "services.0.name", serviceName + "-suffix")
    return res, err
}
```

### Example: Response code modification plugin

This plugin will implement another interface on the plugin from the previous section.
It will modify the response status code for provision operation.

```go
package mypackage

func (p *MyPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
    if !checkCredentials(req) {
        return &web.Response{
            StatusCode: 401 // Unauthorized
        }, nil
    }

    return next(req)
}
```

### Register a plugin

In order to add this plugin to the Service Manager one has to do the following:

```go
...

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    sm := servicemanager.New(ctx, cancel)
    sm.RegisterPlugin(&mypackage.MyPlugin{})
    sm.Run()
}

...
```

## Best practices for writing filters and plugins

### Request and response body modifications

Request and response work with plain byte arrays (usually JSON). That's why it is recommended:

* For JSON modification (as in the [catalog plugin](#catalog-modification-plugin)) use this library https://github.com/tidwall/sjson
* To extract some value from JSON use https://github.com/tidwall/gjson
* **NOTE:** Be aware that JSON request and response may contain non-standard properties.
So when modifying the JSON body make sure to preserve them.
For example avoid marshalling from fixed structures.

### Do not modify the original Request object

`web.Request` contains the original `http.Request` object, but it **should NOT** be modified as this might lead to undesired behaviors.

**Do NOT** use the `http.Request.Body` as this reader is already processed by Service Manager code. The body can be accessed from `web.Request.Body` which is a byte array.

### Chaining

As part of execution of filter/plugin code call the `next(req)` function, this will forward control to the next filter/plugin in the chain.
In case filter/plugin logic requires to stop the chain and exit, just omit the call to `next(req)` function.

### Error handling

In case of error, return `nil` response and an error object.
Use `web.NewHTTPError` function to send error information and status code to the HTTP client.
All other errors will result in status 500 (Internal Server Error) being returned to the client.
