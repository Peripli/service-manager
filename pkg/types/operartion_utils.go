package types

func (e *Operation) IsAsyncResponse() bool {
	return e.Context.Async
}
