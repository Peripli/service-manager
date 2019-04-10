# Interceptors

Interceptors can be attached to CREATE, UPDATE or DELETE storage operations for a specific object type in order to
add additional logic either around the storage transaction or in it.

Interceptors provide  `AroundTx` and `OnTx`. As these operations are chained, you can define your logic in the following places:
1. Before Transaction - Logic is placed inside `AroundTx` before calling the next interceptor in the chain
2. In Transaction Before Operation - Logic is placed inside `OnTx` before calling the next interceptor in the chain
3. In Transaction After Operation - Logic is placed inside `OnTx` after calling the next interceptor in the chain
4. After Transaction - Logic is placed inside `AroundTx` after calling the next interceptor in the chain

## Interceptor Provider

Each interceptor needs its own named provider so that a new interceptor can be provided on each request.
Providers are named, so that you can specify their order.

## Ordering

You can register interceptor providers before or after another interceptor provider.  
Also, you can order independently order the interceptor's `AroundTx` and `OnTx` functions.

### `AroundTx` and `OnTx` ordering

```go
smb.WithCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		Before("anotherCreateInterceptorProvider").
		Register()
```

```go
smb.WithCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		After("anotherCreateInterceptorProvider").
		Register()
```

### `OnTx` only ordering

> **Note** : When specifying only the order of `OnTx`, `AroundTx` will be added as last.

```go
smb.WithCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		OnTxBefore("anotherCreateInterceptorProvider").
		Register()
```

```go
smb.WithCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		OnTxAfter("anotherCreateInterceptorProvider").
		Register()
```

### `AroundTx` only ordering

> **Note** : When specifying only the order of `AroundTx`, `OnTx` will be added as last.

```go
smb.WithCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		AroundTxBefore("anotherCreateInterceptorProvider").
		Register()
```

```go
smb.WithCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		AroundTxAfter("anotherCreateInterceptorProvider").
		Register()
```

### Mixed ordering

```go
smb.WithCreateInterceptorProvider(types.ServiceBrokerType, myCreateInterceptorProvider).
		AroundTxBefore("anotherCreateInterceptorProvider").
		OnTxAfter("anotherCreateInterceptorProvider").
		Register()
```

> **NOTE: Due to the nested chaining of interceptors, registering your interceptor before another interceptor, means that
your post-logic will be executed after the interceptor you've registered before.**

## Examples

### CreateInterceptor

```go
type createInterceptorProvider struct {
}

func (c *createInterceptorProvider) Provide() storage.CreateInterceptor {
	return &CreateInterceptor{}
}
func (c *createInterceptorProvider) Name() string {
	return "CreateInterceptorProvider"
}

type CreateInterceptor struct {
	state    string
}

func (c *CreateInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		// This is Pre-Operation logic
		c.state = "READY"
		return h(ctx, obj)
	}
}

func (c *CreateInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Warehouse, obj types.Object) error {
		// This shows how use data in the transaction which was obtained before the transaction
		if c.state != "READY"{
			return fmt.Errorf("Expected state to be READY before proceeding with transaction. Received: %s", c.state)
		}
		if err := f(ctx, txStorage, obj); err != nil {
			return err
		}
		// Create additional resources in the same transaction after the main resource has been created
		return nil
	}
}

```

### UpdateInterceptor

```go
type updateInterceptorProvider struct {
}

func (c *updateInterceptorProvider) Provide() storage.UpdateInterceptor {
	return &UpdateInterceptor{}
}
func (c *updateInterceptorProvider) Name() string {
	return "UpdateInterceptorProvider"
}

type UpdateInterceptor struct {
}

func (c *UpdateInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		labelChanges = append(labelChanges, &query.LabelChange{
			Key:       "newLabelKey",
			Operation: query.AddLabelOperation,
			Values:    []string{"newLabelValue1"},
		})
		return h(ctx, obj, labelChanges...)
	}
}

func (c *UpdateInterceptor) OnTxUpdate(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		labelChanges = append(labelChanges, &query.LabelChange{
			Key:       "newLabelKey",
			Operation: query.AddLabelOperation,
			Values:    []string{"newLabelValue2"},
		})
		return f(ctx, txStorage, obj, labelChanges)
	}
}

```

### DeleteInterceptor

```go
type deleteInterceptorProvider struct {
}

func (c *deleteInterceptorProvider) Provide() storage.DeleteInterceptor {
	return &DeleteInterceptor{}
}
func (c *deleteInterceptorProvider) Name() string {
	return "DeleteInterceptorProvider"
}

type DeleteInterceptor struct {
}

func (c *DeleteInterceptor) AroundTxDelete(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return h
}

func (c *DeleteInterceptor)OnTxDelete(f storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
	return func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		// Additional operation inside transaction before committing
		return f(ctx, txStorage, deletionCriteria...)
	}
}

```
