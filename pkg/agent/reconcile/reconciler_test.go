/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package reconcile_test

import (
	"context"
	"sync"

	"github.com/Peripli/service-manager/pkg/agent/reconcile"

	"github.com/Peripli/service-manager/pkg/agent/notifications"
	"github.com/Peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler", func() {
	Describe("Process", func() {
		var (
			wg         *sync.WaitGroup
			reconciler *reconcile.Reconciler
			messages   chan *notifications.Message
			ctx        context.Context
			cancel     context.CancelFunc
			resyncer   *fakeResyncer
			consumer   *fakeConsumer
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			wg = &sync.WaitGroup{}
			resyncer = &fakeResyncer{}
			consumer = &fakeConsumer{}

			reconciler = &reconcile.Reconciler{
				Resyncer: resyncer,
				Consumer: consumer,
			}
			messages = make(chan *notifications.Message, 10)
		})

		Context("when the context is canceled", func() {
			It("quits", func(done Done) {
				reconciler.Reconcile(ctx, messages, wg)
				cancel()
				wg.Wait()
				close(done)
			}, 0.1)
		})

		Context("when the messages channel is closed", func() {
			It("quits", func(done Done) {
				reconciler.Reconcile(ctx, messages, wg)
				close(messages)
				wg.Wait()
				close(done)
			}, 0.1)
		})

		Context("when the messages channel is closed after resync", func() {
			It("quits", func(done Done) {
				messages <- &notifications.Message{Resync: true}
				close(messages)
				reconciler.Reconcile(ctx, messages, wg)
				wg.Wait()
				close(done)
			}, 0.1)
		})

		Context("when notifications are sent", func() {
			It("applies them in the same order", func(done Done) {
				ns := []*types.Notification{
					{
						Resource: "/v1/service_brokers",
						Type:     "CREATED",
					},
					{
						Resource: "/v1/service_brokers",
						Type:     "DELETED",
					},
				}
				for _, n := range ns {
					messages <- &notifications.Message{Notification: n}
				}
				close(messages)
				reconciler.Reconcile(ctx, messages, wg)
				wg.Wait()
				Expect(consumer.consumedNotifications).To(Equal(ns))
				close(done)
			})
		})

		Context("when resync is sent", func() {
			It("drops all remaining messages in the queue and processes all new messages", func(done Done) {
				nCreated := &types.Notification{
					Resource: "/v1/service_brokers",
					Type:     "CREATED",
				}
				nDeleted := &types.Notification{
					Resource: "/v1/service_brokers",
					Type:     "DELETED",
				}
				nModified := &types.Notification{
					Resource: "/v1/service_brokers",
					Type:     "MODIFIED",
				}
				messages <- &notifications.Message{Notification: nCreated}
				messages <- &notifications.Message{Resync: true}
				messages <- &notifications.Message{Notification: nDeleted}
				messages <- &notifications.Message{Resync: true}
				reconciler.Reconcile(ctx, messages, wg)

				Eventually(consumer.GetConsumedNotifications).Should(Equal([]*types.Notification{nCreated}))
				Expect(resyncer.GetResyncCount()).To(Equal(1))

				messages <- &notifications.Message{Notification: nModified}
				close(messages)
				wg.Wait()
				Expect(consumer.GetConsumedNotifications()).To(Equal([]*types.Notification{nCreated, nModified}))
				close(done)
			})
		})
	})
})

type fakeResyncer struct {
	mutex       sync.Mutex
	resyncCount int
}

func (r *fakeResyncer) Resync(ctx context.Context) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.resyncCount++
}

func (r *fakeResyncer) GetResyncCount() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.resyncCount
}

type fakeConsumer struct {
	mutex                 sync.Mutex
	consumedNotifications []*types.Notification
}

func (c *fakeConsumer) Consume(ctx context.Context, notification *types.Notification) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.consumedNotifications = append(c.consumedNotifications, notification)
}

func (c *fakeConsumer) GetConsumedNotifications() []*types.Notification {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.consumedNotifications
}
