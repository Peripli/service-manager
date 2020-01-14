package util

import (
	"context"
	"time"
)

// StateContext is a Context which only holds values.
// Its Deadline(), Done() and Err() implementations have
// been stubbed out. Such a StateContext implementation is needed for scenarios
// where a request context needs to be copied and not derived, so as to not
// have to worry about being canceled by it's parent.
type StateContext struct {
	Context context.Context
}

func (sc StateContext) Value(key interface{}) interface{} {
	return sc.Context.Value(key)
}

func (StateContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (StateContext) Done() <-chan struct{} {
	return nil
}

func (StateContext) Err() error {
	return nil
}
