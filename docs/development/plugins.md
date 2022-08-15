# Plugins

Plugins provide means to intercept different OSB calls (provision, deprovision, bind, etc.).
It can modify both the request before it reaches the broker and the response before being sent to the client.

There are several interfaces that the plugin can implement for different OSB API operations.
They can be found `pkg/web/plugin.go`.
For each OSB operation intercepted by the plugin, Service Manager creates a new filter for the respective HTTP endpoint.

## Example: Catalog modification plugin

For example a plugin that modifies the catalog can be written as follows:

```go
package myplugin

import (
    "github.com/tidwall/sjson"
    "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
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

## Example: Response code modification plugin

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
