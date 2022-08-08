package storage

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type cache struct {
	items    map[string]interface{}
	lock     sync.RWMutex
	onResync func() error
	sync     *synchronizer
}

type synchronizer struct {
	Interval time.Duration
	stop     chan bool
}
type Cache struct {
	*cache
}

func NewCache(resyncInterval time.Duration, onResync func() error) *Cache {
	items := make(map[string]interface{})
	c := newCache(onResync, items)
	C := &Cache{c}
	if resyncInterval > 0 {
		runSynchronizer(c, resyncInterval)
		runtime.SetFinalizer(C, StopSynchronizer)

	}
	return C

}
func newCache(onResync func() error, m map[string]interface{}) *cache {
	c := &cache{
		items:    m,
		onResync: onResync,
	}
	return c
}

func StopSynchronizer(c *Cache) {
	c.sync.stop <- true
}

func runSynchronizer(c *cache, ci time.Duration) {
	s := &synchronizer{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.sync = s
	go s.Run(c)
}

func (s *synchronizer) Run(c *cache) {
	ticker := time.NewTicker(s.Interval)
	for {
		select {
		case <-ticker.C:
			c.Resync()
		case <-s.stop:
			ticker.Stop()
			return
		}
	}
}
func (c *cache) Flush() {
	c.items = make(map[string]interface{})
}

func (c *cache) FlushC() {
	defer c.lock.Unlock()
	c.lock.Lock()
	c.items = make(map[string]interface{})

}
func (c *cache) Length() int {
	return len(c.items)
}

func (c *cache) Resync() {
	defer c.lock.Unlock()
	c.lock.Lock()
	c.onResync()

}

func (c *cache) GetC(k string) (interface{}, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, found := c.items[k]
	return item, found
}
func (c *cache) Get(k string) (interface{}, bool) {
	item, found := c.items[k]
	return item, found
}

func (c *cache) Delete(k string) {
	delete(c.items, k)
}
func (c *cache) Add(k string, x interface{}) {
	c.items[k] = x
}

func (c *cache) DeleteC(k string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.items, k)

}

func (c *cache) AddC(k string, x interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, found := c.Get(k)
	if found {
		return fmt.Errorf("Item %s already exists", k)
	}
	c.items[k] = x

	return nil
}
