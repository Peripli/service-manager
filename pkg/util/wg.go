package util

import "sync"

func StartInWaitGroup(f func(), group *sync.WaitGroup) {
	group.Add(1)
	go func() {
		defer group.Done()
		f()
	}()
}
