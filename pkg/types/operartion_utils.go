package types

func (e *Operation) IsAsyncResponse() bool {
	return e.Context.Async
}

func (e *Operation) GetUserInfo() *UserInfo {
	if e.Context != nil {
		return e.Context.UserInfo
	}
	return nil
}
