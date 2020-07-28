package types

func (e *Operation) IsAsyncResponse() bool {

	if e.Context.BrokerResponse.ByBrokerResponse {
		return e.Context.BrokerResponse.Async
	}

	return e.Context.Async
}
