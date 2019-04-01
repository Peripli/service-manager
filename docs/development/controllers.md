# Controllers

Controllers provide means to add additional APIs to the Service Manager. A controller is a way to group a set of routes. Registering a controller in the Service Manager would register the controller routes with their respective handlers as part of the REST API. The registration happens only during Service Manager startup. The Service Manager `pkg/web` package exposes interfaces one should implement in order to add additional SM APIs.

## Example Controller

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
