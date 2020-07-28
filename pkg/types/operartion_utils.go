package types

func (e *Operation) IsAsyncResponse() bool {

	if e.Context.IsAsyncDefinedByClient {
		return e.Context.BrokerResponse.Async
	}

	return e.Context.Async
}
