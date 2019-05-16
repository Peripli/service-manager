/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	notificationConnection "github.com/Peripli/service-manager/storage/postgres/notification_connection"
	notificationConnectionFakes "github.com/Peripli/service-manager/storage/postgres/notification_connection/notification_connectionfakes"

	"github.com/Peripli/service-manager/storage/postgres/postgresfakes"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/lib/pq"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notificator", func() {
	const (
		defaultLastRevision int64 = 10
	)

	var (
		ctx                        context.Context
		cancel                     context.CancelFunc
		wg                         *sync.WaitGroup
		fakeStorage                *postgresfakes.FakeNotificationStorage
		fakeConnectionCreator      *postgresfakes.FakeNotificationConnectionCreator
		testNotificator            storage.Notificator
		fakeNotificationConnection *notificationConnectionFakes.FakeNotificationConnection
		notificationChannel        chan *pq.Notification
		runningFunc                func(isRunning bool, err error)
		queue                      storage.NotificationQueue
		defaultPlatform            *types.Platform
	)

	expectedError := errors.New("*Expected*")

	expectRegisterConsumerFail := func(errorMessage string) {
		q, revision, err := testNotificator.RegisterConsumer(defaultPlatform)
		Expect(q).To(BeNil())
		Expect(revision).To(Equal(types.INVALIDREVISION))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errorMessage))
	}

	expectRegisterConsumerSuccess := func(platform *types.Platform) storage.NotificationQueue {
		q, revision, err := testNotificator.RegisterConsumer(platform)
		Expect(err).ToNot(HaveOccurred())
		Expect(revision).To(Equal(defaultLastRevision))
		Expect(q).ToNot(BeNil())
		return q
	}

	registerDefaultPlatform := func() storage.NotificationQueue {
		return expectRegisterConsumerSuccess(defaultPlatform)
	}

	expectReceivedNotification := func(expectedNotification *types.Notification, q storage.NotificationQueue) {
		receivedNotificationChan := q.Channel()
		Expect(<-receivedNotificationChan).To(Equal(expectedNotification))
	}

	newNotificator := func(queueSize int) storage.Notificator {
		return &Notificator{
			queueSize:       queueSize,
			connectionMutex: &sync.Mutex{},
			consumersMutex:  &sync.Mutex{},
			consumers: &consumers{
				queues:    make(map[string][]storage.NotificationQueue),
				platforms: make([]*types.Platform, 0),
			},
			storage:           fakeStorage,
			connectionCreator: fakeConnectionCreator,
			lastKnownRevision: types.INVALIDREVISION,
		}
	}

	createNotification := func(platformID string) *types.Notification {
		return &types.Notification{
			PlatformID: platformID,
			Revision:   123,
			Type:       "CREATED",
			Resource:   "broker",
			Payload:    json.RawMessage{},
			Base: types.Base{
				ID: "id",
			},
		}
	}

	createNotificationPayload := func(platformID string) string {
		notificationPayload := map[string]interface{}{
			"platform_id":     platformID,
			"notification_id": "notificationID",
			"revision":        defaultLastRevision + 1,
		}
		notificationPayloadJSON, err := json.Marshal(notificationPayload)
		Expect(err).ToNot(HaveOccurred())
		return string(notificationPayloadJSON)
	}

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		wg = &sync.WaitGroup{}
		defaultPlatform = &types.Platform{
			Base: types.Base{
				ID: "platformID",
			},
		}
		fakeStorage = &postgresfakes.FakeNotificationStorage{}
		fakeStorage.GetLastRevisionReturns(defaultLastRevision, nil)
		fakeNotificationConnection = &notificationConnectionFakes.FakeNotificationConnection{}
		fakeNotificationConnection.ListenReturns(nil)
		fakeNotificationConnection.UnlistenReturns(nil)
		fakeNotificationConnection.CloseReturns(nil)
		notificationChannel = make(chan *pq.Notification, 2)
		fakeNotificationConnection.NotificationChannelReturns(notificationChannel)
		runningFunc = nil
		fakeConnectionCreator = &postgresfakes.FakeNotificationConnectionCreator{}
		fakeConnectionCreator.NewConnectionStub = func(f func(isRunning bool, err error)) notificationConnection.NotificationConnection {
			runningFunc = f
			return fakeNotificationConnection
		}
		testNotificator = newNotificator(1)
	})

	AfterEach(func() {
		cancel()
		wg.Wait()
	})

	Describe("Start", func() {

		Context("When already started", func() {
			BeforeEach(func() {
				Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			})

			It("Should return error", func() {
				err := testNotificator.Start(ctx, wg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("notificator already started"))
			})
		})
	})

	Describe("RegisterFilter", func() {

		var queue2 storage.NotificationQueue
		var secondPlatform *types.Platform

		BeforeEach(func() {
			secondPlatform = &types.Platform{
				Base: types.Base{
					ID: "platform2",
				},
			}
			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			runningFunc(true, nil)
			queue = registerDefaultPlatform()
			queue2 = expectRegisterConsumerSuccess(secondPlatform)
			testNotificator.RegisterFilter(func(recipients []*types.Platform, notification *types.Notification) []*types.Platform {
				Expect(recipients).To(HaveLen(2))
				return []*types.Platform{recipients[1]}
			})
		})

		Context("When notification is sent with empty platform ID", func() {
			It("Should be filtered in the second queue", func() {
				notification := createNotification("")
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(notification.PlatformID),
				}
				expectReceivedNotification(notification, queue2)
				Expect(queue.Channel()).To(HaveLen(0))
			})
		})
	})

	Describe("UnregisterConsumer", func() {
		BeforeEach(func() {
			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			Expect(runningFunc).ToNot(BeNil())
			runningFunc(true, nil)
			queue = registerDefaultPlatform()
		})

		newQueue := func(size int) storage.NotificationQueue {
			q, err := storage.NewNotificationQueue(size)
			Expect(err).ToNot(HaveOccurred())
			return q
		}

		Context("When id is not found", func() {
			It("Should return nil", func() {
				q := newQueue(1)
				err := testNotificator.UnregisterConsumer(q)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When id is found", func() {
			It("Should unregister consumer", func() {
				err := testNotificator.UnregisterConsumer(queue)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeNotificationConnection.UnlistenCallCount()).To(Equal(1))
			})
		})

		Context("When more than one consumer is registered", func() {
			It("Should not unlisten", func() {
				registerDefaultPlatform()
				err := testNotificator.UnregisterConsumer(queue)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeNotificationConnection.UnlistenCallCount()).To(Equal(0))
			})
		})

		Context("When unlisten returns error", func() {
			It("Should unregister consumer", func() {
				fakeNotificationConnection.UnlistenReturns(expectedError)
				err := testNotificator.UnregisterConsumer(queue)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedError.Error()))
			})
		})
	})

	Describe("RegisterConsumer", func() {

		BeforeEach(func() {
			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			runningFunc(true, nil)
		})

		Context("When storage GetLastRevision fails", func() {
			BeforeEach(func() {
				fakeStorage.GetLastRevisionReturns(types.INVALIDREVISION, expectedError)
			})

			It("Should return error", func() {
				expectRegisterConsumerFail("listen to notifications channel failed " + expectedError.Error())
			})
		})

		Context("When Notificator is running", func() {
			It("Should not return error", func() {
				registerDefaultPlatform()
				Expect(fakeNotificationConnection.ListenCallCount()).To(Equal(1))
			})
		})

		Context("When Notificator stops", func() {
			It("Should return error", func() {
				registerDefaultPlatform()
				runningFunc(false, nil)
				expectRegisterConsumerFail("cannot register consumer - Notificator is not running")
			})
		})

		Context("When listen returns error", func() {
			It("Should return error", func() {
				fakeNotificationConnection.ListenReturns(expectedError)
				expectRegisterConsumerFail(expectedError.Error())
			})
		})

	})

	Describe("Process notifications", func() {

		BeforeEach(func() {
			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			runningFunc(true, nil)
			queue = registerDefaultPlatform()
		})

		Context("When notification is sent", func() {
			It("Should be received in the queue", func() {
				notification := createNotification(defaultPlatform.ID)
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID),
				}
				expectReceivedNotification(notification, queue)
			})
		})

		Context("When notification cannot be fetched from db", func() {
			fetchNotificationFromDBFail := func(platformID string) {
				fakeStorage.GetNotificationReturns(nil, expectedError)
				ch := queue.Channel()
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(platformID),
				}
				_, ok := <-ch
				Expect(ok).To(BeFalse())
			}

			Context("When notification has registered platform ID", func() {
				It("queue should be closed", func() {
					fetchNotificationFromDBFail(defaultPlatform.ID)
				})
			})

			Context("When notification has empty platform ID", func() {
				It("queue should be closed", func() {
					fetchNotificationFromDBFail("")
				})
			})
		})

		Context("When notification is sent with empty platform ID", func() {
			It("Should be received in the queue", func() {
				q := registerDefaultPlatform()

				notification := createNotification("")
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(""),
				}
				expectReceivedNotification(notification, queue)
				expectReceivedNotification(notification, q)
			})
		})

		Context("When notification is sent with unregistered platform ID", func() {
			It("Should call storage once", func() {
				notification := createNotification(defaultPlatform.ID)
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload("not_registered"),
				}
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID),
				}
				expectReceivedNotification(notification, queue)
				Expect(fakeStorage.GetNotificationCallCount()).To(Equal(1))
			})
		})

		Context("When notification is sent from db with invalid payload", func() {
			It("Should close notification queue", func() {
				ch := queue.Channel()
				notificationChannel <- &pq.Notification{
					Extra: "not_json",
				}
				_, ok := <-ch
				Expect(ok).To(BeFalse())
			})
		})

		Context("When notification is null", func() {
			It("Should not send notification", func() {
				notification := createNotification(defaultPlatform.ID)
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- nil
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID),
				}
				expectReceivedNotification(notification, queue)
			})
		})

		Context("When notification is sent to full queue", func() {

			var notificationChannel chan *pq.Notification

			BeforeEach(func() {
				runningFunc = nil
				testNotificator = newNotificator(0)
				Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
				Expect(runningFunc).ToNot(BeNil())
				notificationChannel = make(chan *pq.Notification, 2)
				fakeNotificationConnection.NotificationChannelReturns(notificationChannel)
				runningFunc(true, nil)
			})

			It("Should close notification queue", func() {
				q := registerDefaultPlatform()
				notification := createNotification(defaultPlatform.ID)
				fakeStorage.GetNotificationReturns(notification, nil)
				ch := q.Channel()
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID),
				}
				_, ok := <-ch
				Expect(ok).To(BeFalse())
			})
		})
	})
})
