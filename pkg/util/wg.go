package util

import (
	"context"
	"sync"
)

func StartInWaitGroup(f func(), group *sync.WaitGroup) {
	group.Add(1)
	go func() {
		defer group.Done()
		f()
	}()
}

func StartInWaitGroupWithContext(ctx context.Context, f func(ctx context.Context), group *sync.WaitGroup) {
	group.Add(1)
	go func() {
		defer group.Done()
		f(ctx)
	}()
}
