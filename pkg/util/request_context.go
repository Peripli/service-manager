package util

import (
	"context"
	"time"
)

// RequestContextCopy is a Context which only holds values.
// It's Deadline(), Done() and Err() implementations have
// been stubbed out. Such a RequestContextCopy implementation is needed for scenarios
// where a request context needs to be copied and not derived and thus not
// have to worry about being canceled by it's parent.
type RequestContextCopy struct {
	Context context.Context
}

func (rcc RequestContextCopy) Value(key interface{}) interface{} {
	return rcc.Context.Value(key)
}

func (RequestContextCopy) Deadline() (deadline time.Time, ok bool) {
	return
}

func (RequestContextCopy) Done() <-chan struct{} {
	return nil
}

func (RequestContextCopy) Err() error {
	return nil
}
