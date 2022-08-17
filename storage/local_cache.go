package storage

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type cache struct {
	items         map[string]interface{}
	lock          sync.RWMutex
	onTimeExpired func() error
	onFlush       func() error
	sync          *janitor
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}
type Cache struct {
	*cache
}

func NewCache(junitorInterval time.Duration, onTimeExpired func() error, onFlush func() error) *Cache {
	items := make(map[string]interface{})
	c := newCache(onTimeExpired, onFlush, items)
	C := &Cache{c}
	if junitorInterval > 0 {
		runJanitor(c, junitorInterval)
		runtime.SetFinalizer(C, StopSynchronizer)

	}
	return C

}
func newCache(onTimeExpired func() error, onFlush func() error, m map[string]interface{}) *cache {
	c := &cache{
		items:         m,
		onTimeExpired: onTimeExpired,
		onFlush:       onFlush,
	}
	return c
}

func StopSynchronizer(c *Cache) {
	c.sync.stop <- true
}

func runJanitor(c *cache, ci time.Duration) {
	s := &janitor{
		Interval: ci,
		stop:     make(chan bool),
	}
	c.sync = s
	go s.Run(c)
}

func (s *janitor) Run(c *cache) {
	ticker := time.NewTicker(s.Interval)
	for {
		select {
		case <-ticker.C:
			c.TimeExpired()
		case <-s.stop:
			ticker.Stop()
			return
		}
	}
}
func (c *cache) Flush() {
	c.items = make(map[string]interface{})
}

func (c *cache) FlushL() error {
	defer c.lock.Unlock()
	c.lock.Lock()
	c.items = make(map[string]interface{})
	if c.onFlush != nil {
		err := c.onFlush()
		if err != nil {
			return fmt.Errorf("error executing onFlush function: %s", err)
		}
	}
	return nil
}
func (c *cache) Length() int {
	return len(c.items)
}

func (c *cache) TimeExpired() {
	defer c.lock.Unlock()
	c.lock.Lock()
	c.onTimeExpired()

}

func (c *cache) Get(k string) (interface{}, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, found := c.items[k]
	return item, found
}
func (c *cache) get(k string) (interface{}, bool) {
	item, found := c.items[k]
	return item, found
}

func (c *cache) delete(k string) {
	delete(c.items, k)
}
func (c *cache) add(k string, x interface{}) {
	c.items[k] = x
}

func (c *cache) Delete(k string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.items, k)

}

func (c *cache) Add(k string, x interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, found := c.Get(k)
	if found {
		return fmt.Errorf("Item %s already exists", k)
	}
	c.items[k] = x

	return nil
}
