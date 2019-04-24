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
	"fmt"
	"sync"
	"testing"

	notificationStorage "github.com/Peripli/service-manager/notifications/postgres/storage"
	"github.com/Peripli/service-manager/notifications/postgres/storage/storagefakes"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/lib/pq"

	"github.com/Peripli/service-manager/notifications"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNotificator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Postgres Notifications Suite")
}

var _ = Describe("Notificator", func() {
	const (
		defaultLastRevision int64 = 10
		invalidRevision     int64 = -1
		defaultPlatformID         = "platformID"
	)

	var (
		ctx                        context.Context
		cancel                     context.CancelFunc
		wg                         *sync.WaitGroup
		fakeStorage                *storagefakes.FakeNotificationStorage
		testNotificator            notifications.Notificator
		fakeNotificationConnection *storagefakes.FakeNotificationConnection
		notificationChannel        chan *pq.Notification
		runningFunc                func(isRunning bool, err error)
		userContext                *web.UserContext
		fakeData                   *webfakes.FakeData
		queue                      notifications.NotificationQueue
	)

	expectedError := errors.New("*Expected*")

	expectRegisterConsumerFail := func(errorMessage string) {
		q, revision, err := testNotificator.RegisterConsumer(userContext)
		Expect(q).To(BeNil())
		Expect(revision).To(Equal(invalidRevision))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errorMessage))
	}

	expectRegisterConsumerSuccess := func() notifications.NotificationQueue {
		q, revision, err := testNotificator.RegisterConsumer(userContext)
		Expect(err).ToNot(HaveOccurred())
		Expect(revision).To(Equal(defaultLastRevision))
		Expect(q).ToNot(BeNil())
		return q
	}

	expectReceivedNotification := func(expectedNotification *types.Notification, q notifications.NotificationQueue) {
		receivedNotificationChan := q.Channel()
		Expect(<-receivedNotificationChan).To(Equal(expectedNotification))
	}

	newNotificator := func(queueSize int) notifications.Notificator {
		return &notificator{
			queueSize:         queueSize,
			isRunningMutex:    &sync.RWMutex{},
			connectionMutex:   &sync.Mutex{},
			consumersMutex:    &sync.RWMutex{},
			consumers:         make(consumers),
			storage:           fakeStorage,
			lastKnownRevision: invalidRevisionNumber,
			revisionMutex:     &sync.RWMutex{},
		}
	}

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		wg = &sync.WaitGroup{}
		fakeStorage = &storagefakes.FakeNotificationStorage{}
		fakeStorage.GetLastRevisionReturns(defaultLastRevision, nil)
		fakeNotificationConnection = &storagefakes.FakeNotificationConnection{}
		fakeNotificationConnection.ListenReturns(nil)
		fakeNotificationConnection.UnlistenReturns(nil)
		fakeNotificationConnection.CloseReturns(nil)
		notificationChannel = make(chan *pq.Notification, 2)
		fakeNotificationConnection.NotificationChannelReturns(notificationChannel)
		runningFunc = nil
		fakeStorage.NewConnectionStub = func(f func(isRunning bool, err error)) notificationStorage.NotificationConnection {
			runningFunc = f
			return fakeNotificationConnection
		}
		testNotificator = newNotificator(1)
		fakeData = &webfakes.FakeData{}
		fakeData.DataStub = func(i interface{}) error {
			platform := i.(*types.Platform)
			platform.ID = defaultPlatformID
			return nil
		}
		userContext = &web.UserContext{}
		userContext.Data = fakeData
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

	Describe("UnregisterConsumer", func() {
		BeforeEach(func() {
			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			Expect(runningFunc).ToNot(BeNil())
			runningFunc(true, nil)
			queue = expectRegisterConsumerSuccess()
		})

		newQueue := func(size int) notifications.NotificationQueue {
			q, err := notifications.NewNotificationQueue(size)
			Expect(err).ToNot(HaveOccurred())
			return q
		}

		Context("When id is not found", func() {
			It("Should return error", func() {
				q := newQueue(1)
				err := testNotificator.UnregisterConsumer(q)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("consumer %s was not found", q.ID())))
			})
		})

		Context("When id is found", func() {
			It("Should unregister consumer", func() {
				err := testNotificator.UnregisterConsumer(queue)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeNotificationConnection.UnlistenCallCount()).To(Equal(1))
				err = testNotificator.UnregisterConsumer(queue)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("consumer %s was not found", queue.ID())))
			})
		})

		Context("When more than one consumer is registered", func() {
			It("Should not unlisten", func() {
				expectRegisterConsumerSuccess()
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
				fakeStorage.GetLastRevisionReturns(invalidRevision, expectedError)
			})

			It("Should return error", func() {
				expectRegisterConsumerFail("listen to notifications channel failed " + expectedError.Error())
			})
		})

		Context("When user is not valid", func() {
			It("Should return error", func() {
				fakeData.DataReturns(expectedError)
				expectRegisterConsumerFail("could not get platform from user context " + expectedError.Error())
			})
		})

		Context("When user is empty", func() {
			It("Should return error", func() {
				fakeData.DataStub = func(i interface{}) error {
					return nil
				}
				expectRegisterConsumerFail("platform ID not found in user context")
			})
		})

		Context("When notificator is running", func() {
			It("Should not return error", func() {
				expectRegisterConsumerSuccess()
				Expect(fakeNotificationConnection.ListenCallCount()).To(Equal(1))
			})
		})

		Context("When notificator stops", func() {
			It("Should return error", func() {
				expectRegisterConsumerSuccess()
				runningFunc(false, nil)
				expectRegisterConsumerFail("cannot register consumer - notificator is not running")
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
			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			runningFunc(true, nil)
			queue = expectRegisterConsumerSuccess()
		})

		Context("When notification is sent", func() {
			It("Should be received in the queue", func() {
				notification := createNotification(defaultPlatformID)
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatformID),
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
					fetchNotificationFromDBFail(defaultPlatformID)
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
				q := expectRegisterConsumerSuccess()

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
				notification := createNotification(defaultPlatformID)
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload("not_registered"),
				}
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatformID),
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
				notification := createNotification(defaultPlatformID)
				fakeStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- nil
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatformID),
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
				q := expectRegisterConsumerSuccess()
				notification := createNotification(defaultPlatformID)
				fakeStorage.GetNotificationReturns(notification, nil)
				ch := q.Channel()
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatformID),
				}
				_, ok := <-ch
				Expect(ok).To(BeFalse())
			})
		})
	})
})
