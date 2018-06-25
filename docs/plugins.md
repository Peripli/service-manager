# Plugins

Plugins provide means to intercept different OSB calls (provision, deprovision, bind, etc.).
One can modify both the request before it reaches the broker and the response before being sent to the client.

There are several interfaces that the plugin can implement for different OSB API operations.
They can be found [here](pkg/plugin/osb.go)

There are several types one has to know about. They can be found [here](pkg/filter/filter.go)

## Catalog modification plugin

For example a plugin that modifies the catalog can be written as follows:

```go
package mypackage

import (
    "github.com/tidwall/sjson"
)

type MyPlugin struct {}

func (p *MyPlugin) FetchCatalog(req *filter.Request, next filter.Handler) (*filter.Response, error) {
    res, err := next(req)
    if err != nil {
        return nil, err
    }
    serviceName := gjson.GetBytes(res.Body, "services.0.name").String()
    res.Body, err = sjson.SetBytes(res.Body, "services.0.name", serviceName + "-suffix")
    return res, err
}
```

## Response code modification plugin

This plugin will implement another interface on the plugin from the previous section.
It will modify the response status code for provision operation.

```go
package mypackage

import (
    "github.com/tidwall/sjson"
)

func (p *MyPlugin) Provision(req *filter.Request, next filter.Handler) (*filter.Response, error) {
    if !checkCredentials(req) {
        return &filter.Response{
            StatusCode: 401 // Unauthorized
        }, nil
    }

    return next(req)
}
```

## Use the plugin

In order to add this plugin to the Service Manager one has to do the following:

```go
...

func main() {
    flags := initFlags()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    handleInterrupts(ctx, cancel)

    api := &rest.API{}
    // Register the plugin in the API
    api.RegisterPlugins(MyPlugin{})

    config := &sm.Parameters{
        Context:     ctx,
        Environment: getEnvironment(flags),
        // Provide the extended API
        API:         api,
    }
    srv, err := sm.NewServer(config)
    if err != nil {
        logrus.Fatal("Error creating the server: ", err)
    }

    srv.Run(ctx)
}

...
```

## Best practices in writing plugins

### Request and response body modifications

Request and response work with plain byte arrays (usually JSON). That's why we recommend:

* For JSON modification (as in the [catalog plugin](#catalog-modification-plugin)) use this library https://github.com/tidwall/sjson
* To extract some value from JSON use https://github.com/tidwall/gjson

### Do not modify the original Request object

Our `filter.Request` contains the original `http.Request` object, but it **should NOT** be modified as this might lead to undesired behaviours.

**Do NOT** use the `http.Request.Body` as this reader is already processed by our code. The body can be accessed from `filter.Request.Body` which is a byte array.

### Chaining

As part of execution of your code call the `next(req)` function, this will forward control to the next plugin in the chain.
In case your logic require to stop the chain and exit, just omit the call to next(req) function.
