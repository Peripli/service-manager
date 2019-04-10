# Extending the Service Manager

The main extension points of the service manager are filters, plugins, controllers and interceptors. The
interfaces that need to be implemented in order to provide an extension point can be found
in `pkg/web`, `storage/interceptors_create.go`, `storage/interceptors_update.go` and `storage/interceptors_update.go`.

In addition, for your components you can provide your own health metrics by adding a simple health indicator.
You can also configure how all the provided health metrics be displayed by registering your aggregation policy
which can customize the health report.

## Extension types

- [Filters](./filters.md)
- [Plugins](./plugins.md)
- [Controllers](./controllers.md)
- [Interceptors](./interceptors.md)
- [Health](./health.md)

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
    serviceManager.
    	WithCreateInterceptorProvider(types.PlatformType, &myinterceptor.MyInterceptorProvider{}).
    	Register()

    sm := serviceManager.Build()
    sm.Run()
}
...
```

## Extensions in the Service Broker Proxies

The service broker proxies (currently the [K8S proxy](https://github.com/Peripli/service-broker-proxy-k8s) and 
the [CF proxy](httyps://github.com/Peripli/service-broker-proxy-cf)) that base their implementation on 
the [Broker Proxy Framework](https://github.com/Peripli/service-broker-proxy) by default get the same extension 
capabilities that the Service Manager has except interceptors. This would imply that one can register filters,
plugins, controllers and health indicators as extensions in the proxies, too.
