# Interceptors

Interceptors can be attached to POST, PATCH or DELETE requests for a specific object type in order to
add additional logic on the API layer and/or on the storage layer (inside an open transaction).

## Interceptor Provider

Each interceptor needs its own provider so that a new interceptor can be provided on each request.
Providers are named, so that you can specify their order.

## Ordering

You can register interceptor providers before or after another interceptor provider.  
Also, you can order independently the interceptor's `Tx` and `API` functions 

## Registration


## Examples

### CreateInterceptor

Example:

* 