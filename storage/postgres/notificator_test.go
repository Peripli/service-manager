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
	"time"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/util"

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
		defaultQueueSize    int   = 1
	)

	var (
		ctx                        context.Context
		cancel                     context.CancelFunc
		wg                         *sync.WaitGroup
		fakeNotificationStorage    *postgresfakes.FakeNotificationStorage
		fakeConnectionCreator      *postgresfakes.FakeNotificationConnectionCreator
		testNotificator            storage.Notificator
		fakeNotificationConnection *notificationConnectionFakes.FakeNotificationConnection
		notificationChannel        chan *pq.Notification
		runningFunc                func(isRunning bool, err error)
		queue                      storage.NotificationQueue
		defaultPlatform            *types.Platform
	)

	expectedError := errors.New("*Expected*")

	expectRegisterConsumerFail := func(errorMessage string, revision int64) {
		q, smRevision, err := testNotificator.RegisterConsumer(defaultPlatform, revision)
		Expect(q).To(BeNil())
		Expect(smRevision).To(Equal(types.InvalidRevision))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errorMessage))
	}

	expectRegisterConsumerSuccess := func(platform *types.Platform, revision int64) storage.NotificationQueue {
		q, smRevision, err := testNotificator.RegisterConsumer(platform, revision)
		Expect(err).ToNot(HaveOccurred())
		Expect(smRevision).To(Equal(defaultLastRevision))
		Expect(q).ToNot(BeNil())
		return q
	}

	registerDefaultPlatform := func() storage.NotificationQueue {
		return expectRegisterConsumerSuccess(defaultPlatform, types.InvalidRevision)
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
			storage:           fakeNotificationStorage,
			connectionCreator: fakeConnectionCreator,
			stopProcessing:    func() {},
			lastKnownRevision: types.InvalidRevision,
			dbPingInterval:    time.Millisecond * 10,
		}
	}

	createNotification := func(platformID string) *types.Notification {
		id, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		return &types.Notification{
			PlatformID: platformID,
			Revision:   123,
			Type:       "CREATED",
			Resource:   "broker",
			Payload:    json.RawMessage{},
			Base: types.Base{
				ID: id.String(),
			},
		}
	}

	createNotificationPayload := func(platformID, notificationID string) string {
		notificationPayload := map[string]interface{}{
			"platform_id":     platformID,
			"notification_id": notificationID,
			"revision":        defaultLastRevision + 1,
		}
		notificationPayloadJSON, err := json.Marshal(notificationPayload)
		Expect(err).ToNot(HaveOccurred())
		return string(notificationPayloadJSON)
	}

	expectUnlistenCalled := func(unlistenCalled chan struct{}) {
		select {
		case <-unlistenCalled:
			break
		case <-time.After(time.Second * 3):
			Fail("Expected unlisten to be called")
		}
	}

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		wg = &sync.WaitGroup{}
		defaultPlatform = &types.Platform{
			Base: types.Base{
				ID:    "platformID",
				Ready: true,
			},
		}
		fakeNotificationStorage = &postgresfakes.FakeNotificationStorage{}
		fakeNotificationStorage.GetLastRevisionReturns(defaultLastRevision, nil)
		fakeNotificationConnection = &notificationConnectionFakes.FakeNotificationConnection{}
		fakeNotificationConnection.ListenReturns(nil)
		fakeNotificationConnection.UnlistenReturns(nil)
		fakeNotificationConnection.PingReturns(nil)
		fakeNotificationConnection.CloseReturns(nil)
		notificationChannel = make(chan *pq.Notification, 2)
		fakeNotificationConnection.NotificationChannelReturns(notificationChannel)
		runningFunc = nil
		fakeConnectionCreator = &postgresfakes.FakeNotificationConnectionCreator{}
		fakeConnectionCreator.NewConnectionStub = func(f func(isRunning bool, err error)) notificationConnection.NotificationConnection {
			runningFunc = f
			return fakeNotificationConnection
		}
		testNotificator = newNotificator(defaultQueueSize)
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
		var registerWithRevision int64
		var notification *types.Notification

		JustBeforeEach(func() {
			secondPlatform = &types.Platform{
				Base: types.Base{
					ID:    "platform2",
					Ready: true,
				},
			}
			testNotificator.RegisterFilter(func(recipients []*types.Platform, notification *types.Notification) []*types.Platform {
				switch len(recipients) {
				case 1:
					if recipients[0].ID == defaultPlatform.ID {
						return nil
					}
					return recipients
				case 2:
					return []*types.Platform{recipients[1]} // filter the default platform
				default:
					Fail("The registered test filter was called with invalid recipients")
					return nil
				}
			})
			notification = createNotification("")
			fakeNotificationStorage.GetNotificationByRevisionReturns(notification, nil)
			fakeNotificationStorage.ListNotificationsReturns([]*types.Notification{notification}, nil)
			fakeNotificationStorage.GetNotificationReturns(notification, nil)

			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			runningFunc(true, nil)
			queue = expectRegisterConsumerSuccess(defaultPlatform, registerWithRevision)
			queue2 = expectRegisterConsumerSuccess(secondPlatform, registerWithRevision)
		})

		Context("When notification is sent with empty platform ID", func() {
			BeforeEach(func() {
				registerWithRevision = types.InvalidRevision
			})

			It("Should be filtered in the first queue", func() {
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(notification.PlatformID, notification.ID),
				}
				expectReceivedNotification(notification, queue2)
				Expect(queue.Channel()).To(HaveLen(0))
			})
		})

		Context("When old notification is present", func() {
			BeforeEach(func() {
				registerWithRevision = defaultLastRevision - 1
			})

			It("Should be filtered in the first queue", func() {
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
				unlistenCalled := make(chan struct{}, 1)
				fakeNotificationConnection.UnlistenStub = func(s string) error {
					Expect(s).To(Equal(postgresChannel))
					unlistenCalled <- struct{}{}
					return nil
				}
				err := testNotificator.UnregisterConsumer(queue)
				Expect(err).ToNot(HaveOccurred())
				expectUnlistenCalled(unlistenCalled)
			})
		})

		Context("When unregister is called twice on a given consumer", func() {
			It("Should not return error on second unregister", func() {
				err := testNotificator.UnregisterConsumer(queue)
				Expect(err).ToNot(HaveOccurred())
				err = testNotificator.UnregisterConsumer(queue)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When consumer is unregistered and then registered again after unlisten", func() {
			It("Should listen again", func(done Done) {
				fakeNotificationConnection.UnlistenStub = func(s string) error {
					Expect(s).To(Equal(postgresChannel))
					fakeNotificationConnection.UnlistenReturns(nil)
					go func() {
						registerDefaultPlatform()
						Expect(fakeNotificationConnection.ListenCallCount()).To(Equal(2))
						close(done)
					}()
					return nil
				}
				err := testNotificator.UnregisterConsumer(queue)
				Expect(err).ToNot(HaveOccurred())
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

	})

	Describe("RegisterConsumer", func() {

		BeforeEach(func() {
			Expect(testNotificator.Start(ctx, wg)).ToNot(HaveOccurred())
			runningFunc(true, nil)
		})

		Context("When storage GetLastRevision fails", func() {
			BeforeEach(func() {
				fakeNotificationStorage.GetLastRevisionReturns(types.InvalidRevision, expectedError)
			})

			It("Should return error", func() {
				expectRegisterConsumerFail("getting last revision failed "+expectedError.Error(), types.InvalidRevision)
			})
		})

		Context("When Notificator is running", func() {
			It("Should not return error", func() {
				registerDefaultPlatform()
				Expect(fakeNotificationConnection.ListenCallCount()).To(Equal(1))
			})
		})

		Context("When Notificator is running", func() {
			It("Should ping db regularly", func(done Done) {
				fakeNotificationConnection.PingStub = func() error {
					defer close(done)
					return nil
				}
				registerDefaultPlatform()
			})
		})

		Context("When notification revision is grater than the the one SM knows", func() {
			It("Should return error", func() {
				expectRegisterConsumerFail(util.ErrInvalidNotificationRevision.Error(), defaultLastRevision+1)
			})
		})

		Context("When registering with 0 < revision < sm_revision", func() {
			Context("When storage returns error when getting notification with revision", func() {
				It("Should return the error", func() {
					unlistenCalled := make(chan struct{}, 1)
					fakeNotificationConnection.UnlistenStub = func(s string) error {
						Expect(s).To(Equal(postgresChannel))
						unlistenCalled <- struct{}{}
						return nil
					}
					fakeNotificationStorage.GetNotificationByRevisionReturns(nil, expectedError)
					expectRegisterConsumerFail(expectedError.Error(), defaultLastRevision-1)
					expectUnlistenCalled(unlistenCalled)
				})
			})

			Context("When storage returns error and unlisten returns error when getting notification with revision", func() {
				It("Should return the storage error", func() {
					unlistenCalled := make(chan struct{}, 1)
					fakeNotificationConnection.UnlistenStub = func(s string) error {
						Expect(s).To(Equal(postgresChannel))
						unlistenCalled <- struct{}{}
						return errors.New("unlisten error")
					}
					fakeNotificationStorage.GetNotificationByRevisionReturns(nil, expectedError)
					expectRegisterConsumerFail(expectedError.Error(), defaultLastRevision-1)
					expectUnlistenCalled(unlistenCalled)
				})
			})

			Context("When storage returns \"not found\" error when getting notification with revision", func() {
				It("Should return ErrInvalidNotificationRevision", func() {
					fakeNotificationStorage.GetNotificationByRevisionReturns(nil, util.ErrNotFoundInStorage)
					expectRegisterConsumerFail(util.ErrInvalidNotificationRevision.Error(), defaultLastRevision-1)
				})
			})

			Context("When storage returns error on notification list", func() {
				It("Should return the error", func() {
					fakeNotificationStorage.ListNotificationsReturns(nil, expectedError)
					expectRegisterConsumerFail(expectedError.Error(), defaultLastRevision-1)
				})
			})

			Context("When storage returns too many notifications a queue can handle", func() {
				It("Should return ErrInvalidNotificationRevision", func() {
					notificationsToReturn := make([]*types.Notification, 0, defaultQueueSize+1)
					for i := 0; i < defaultQueueSize+1; i++ {
						notificationsToReturn = append(notificationsToReturn, createNotification(""))
					}
					fakeNotificationStorage.ListNotificationsReturns(notificationsToReturn, nil)
					expectRegisterConsumerFail(util.ErrInvalidNotificationRevision.Error(), defaultLastRevision-1)
				})
			})

			Context("When storage returns a missed notification", func() {
				It("Should be in the returned queue", func() {
					n1 := createNotification("")
					n2 := createNotification("")
					fakeNotificationStorage.GetNotificationByRevisionReturns(n1, nil)
					fakeNotificationStorage.ListNotificationsReturns([]*types.Notification{n1}, nil)
					fakeNotificationStorage.GetNotificationReturns(n2, nil)
					queue = expectRegisterConsumerSuccess(defaultPlatform, defaultLastRevision-1)
					queueChannel := queue.Channel()
					Expect(<-queueChannel).To(Equal(n1))
					notificationChannel <- &pq.Notification{
						Extra: createNotificationPayload("", n2.ID),
					}
					Expect(<-queueChannel).To(Equal(n2))
				})
			})
		})

		Context("When Notificator stops", func() {
			It("Should return error", func() {
				registerDefaultPlatform()
				runningFunc(false, nil)
				expectRegisterConsumerFail("cannot register consumer - Notificator is not running", types.InvalidRevision)
			})
		})

		Context("When listen returns error", func() {
			It("Should return error", func() {
				fakeNotificationConnection.ListenReturns(expectedError)
				expectRegisterConsumerFail(expectedError.Error(), types.InvalidRevision)
			})

			Context("And this error is pq.ErrChannelAlreadyOpen", func() {
				It("Should register consumer", func() {
					fakeNotificationConnection.ListenReturns(pq.ErrChannelAlreadyOpen)
					expectRegisterConsumerSuccess(defaultPlatform, types.InvalidRevision)
				})
			})
		})

		Context("When revision is not valid", func() {
			It("Should return error", func() {
				expectRegisterConsumerFail(util.ErrInvalidNotificationRevision.Error(), 987654321)
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
				fakeNotificationStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID, notification.ID),
				}
				expectReceivedNotification(notification, queue)
			})
		})

		Context("When notification cannot be fetched from db", func() {
			fetchNotificationFromDBFail := func(platformID string) {
				fakeNotificationStorage.GetNotificationReturns(nil, expectedError)
				ch := queue.Channel()
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(platformID, "some_id"),
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
				fakeNotificationStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload("", notification.ID),
				}
				expectReceivedNotification(notification, queue)
				expectReceivedNotification(notification, q)
			})
		})

		Context("When notification is sent with unregistered platform ID", func() {
			It("Should call storage once", func() {
				notification := createNotification(defaultPlatform.ID)
				fakeNotificationStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload("not_registered", "some_id"),
				}
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID, notification.ID),
				}
				expectReceivedNotification(notification, queue)
				Expect(fakeNotificationStorage.GetNotificationCallCount()).To(Equal(1))
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
				fakeNotificationStorage.GetNotificationReturns(notification, nil)
				notificationChannel <- nil
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID, notification.ID),
				}
				expectReceivedNotification(notification, queue)
			})
		})

		Context("When notification is sent to full queue", func() {

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
				fakeNotificationStorage.GetNotificationReturns(notification, nil)
				ch := q.Channel()
				notificationChannel <- &pq.Notification{
					Extra: createNotificationPayload(defaultPlatform.ID, notification.ID),
				}
				_, ok := <-ch
				Expect(ok).To(BeFalse())
			})
		})
	})
})
