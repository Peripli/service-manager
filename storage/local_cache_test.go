package storage

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sync"
	"time"
)

var _ = Describe("local cache", func() {
	var localCache *Cache
	var wg *sync.WaitGroup
	BeforeEach(func() {
		wg = &sync.WaitGroup{}

	})
	Context("cache with no resync", func() {
		BeforeEach(func() {
			localCache = NewCache(0, nil)
		})

		Context("cache add objects and flush", func() {
			It("adds and deletes objects", func() {
				wg.Add(1)
				addStrings := func() {
					for i := 1; i < 3; i++ {
						localCache.AddL(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
					}
					wg.Done()
				}
				go addStrings()
				for i := 3; i < 5; i++ {
					localCache.AddL(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
				}
				wg.Wait()
				for i := 1; i < 5; i++ {
					val, _ := localCache.Get(fmt.Sprintf("key-%d", i))
					Expect(val.(string)).To(Equal(fmt.Sprintf("value-%d", i)))
				}
				localCache.FlushL()
				Expect(localCache.Length()).To(BeZero())
			})

		})

	})

	Context("cache with resync", func() {
		var resyncFunc func() error
		BeforeEach(func() {
			resyncFunc = func() error {
				localCache.Flush()
				localCache.Add("0", "new")
				return nil
			}
			localCache = NewCache(time.Second*5, resyncFunc)
		})
		It("should have only new object", func() {
			for i := 1; i < 3; i++ {
				localCache.AddL(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
			}
			time.Sleep(time.Second * 6)
			StopSynchronizer(localCache)
			val, _ := localCache.Get("0")
			Expect(val.(string)).To(Equal("new"))
			Expect(localCache.Length()).To(Equal(1))

		})
	})
})
