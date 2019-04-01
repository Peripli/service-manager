# Provide your own health indicators

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
