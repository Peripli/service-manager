# Interceptors

Interceptors can be attached to POST, PATCH or DELETE requests for a specific object type in order to
add additional logic on the API layer and/or on the storage layer (inside an open transaction).

## Interceptor Provider

Each interceptor needs its own provider so that a new interceptor can be provided on each request.
Providers are named, so that you can specify their order.

## Ordering

You can register interceptor providers before or after another interceptor provider.  
Also, you can order independently the interceptor's `Tx` and `API` functions.

### Order during registration

#### `Tx` and `API` ordering

```go
api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		Before("anotherCreateInterceptorProvider").
		Apply()
```

```go
api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		After("anotherCreateInterceptorProvider").
		Apply()
```

#### `Tx` only ordering

> **Note** : When specifying only the order of `Tx`, `API` will be added as last.

```go
api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		TxBefore("anotherCreateInterceptorProvider").
		Apply()
```

```go
api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		TxAfter("anotherCreateInterceptorProvider").
		Apply()
```

#### `API` only ordering

> **Note** : When specifying only the order of `API`, `Tx` will be added as last.

```go
api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		APIBefore("anotherCreateInterceptorProvider").
		Apply()
```

```go
api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		APIAfter("anotherCreateInterceptorProvider").
		Apply()
```

#### Mixed ordering

```go
api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		APIBefore("anotherCreateInterceptorProvider").
		TxAfter("anotherCreateInterceptorProvider").
		Apply()
```

### Order implementing interface
```go
func (interceptorProvider *MyInterceptorProvider) PositionTx() (PositionType, string) {
	return PositionBefore, "anotherInterceptorProvider"
}

func (interceptorProvider *MyInterceptorProvider) PositionAPI() (PositionType, string) {
	return PositionAfter, "anotherInterceptorProvider"
}

...

api.RegisterCreateInterceptorProvider(types.ServiceBrokerType, &MyInterceptorProvider{}).Apply()
```

## Examples

### CreateInterceptor

```go
type createInterceptorProvider struct {
}

func (c *createInterceptorProvider) Provide() extension.CreateInterceptor {
	return &CreateInterceptor{}
}
func (c *createInterceptorProvider) Name() string {
	return "CreateBrokerInterceptorProvider"
}

type CreateInterceptor struct {
	state    string
}

func (c *CreateInterceptor) OnAPICreate(h extension.InterceptCreateOnAPI) extension.InterceptCreateOnAPI {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		c.state = "READY"
		return h(ctx, obj)
	}
}

func (c *CreateInterceptor) OnTxCreate(f extension.InterceptCreateOnTx) extension.InterceptCreateOnTx {
	return func(ctx context.Context, txStorage storage.Warehouse, obj types.Object) error {
		if c.state != "READY"{
			return fmt.Errorf("Expected state to be READY before proceeding with transaction. Received: %s", c.state)
		}
		return f(ctx, txStorage, obj)
	}
}

```

### UpdateInterceptor

```go
type updateInterceptorProvider struct {
}

func (c *updateInterceptorProvider) Provide() extension.UpdateInterceptor {
	return &UpdateInterceptor{}
}
func (c *updateInterceptorProvider) Name() string {
	return "UpdateInterceptorProvider"
}

type UpdateInterceptor struct {
}

func (c *UpdateInterceptor) OnAPIUpdate(h extension.InterceptUpdateOnAPI) extension.InterceptUpdateOnAPI {
	return func(ctx context.Context, changes *extension.UpdateContext) (object types.Object, e error) {
		changes.LabelChanges = append(changes.LabelChanges, &query.LabelChange{
			Key:       "newLabelKey",
			Operation: query.AddLabelOperation,
			Values:    []string{"newLabelValue1"},
		})
		return h(ctx, changes)
	}
}

func (c *UpdateInterceptor) OnTxUpdate(f extension.InterceptUpdateOnTx) extension.InterceptUpdateOnTx {
	return func(ctx context.Context, txStorage storage.Warehouse, oldObject types.Object, changes *extension.UpdateContext) (object types.Object, e error) {
		changes.LabelChanges = append(changes.LabelChanges, &query.LabelChange{
			Key:       "newLabelKey",
			Operation: query.AddLabelOperation,
			Values:    []string{"newLabelValue2"},
		})
		return f(ctx, txStorage, oldObject, changes)
	}
}

```

### DeleteInterceptor

```go
type deleteInterceptorProvider struct {
}

func (c *deleteInterceptorProvider) Provide() extension.DeleteInterceptor {
	return &DeleteInterceptor{}
}
func (c *deleteInterceptorProvider) Name() string {
	return "DeleteInterceptorProvider"
}

type DeleteInterceptor struct {
}

func (c *DeleteInterceptor) OnAPIDelete(h extension.InterceptDeleteOnAPI) extension.InterceptDeleteOnAPI {
	return h
}

func (c *DeleteInterceptor) OnTxDelete(f extension.InterceptDeleteOnTx) extension.InterceptDeleteOnTx {
	return func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
		// Additional operation inside transaction before committing
		return f(ctx, txStorage, deletionCriteria...)
	}
}

```
