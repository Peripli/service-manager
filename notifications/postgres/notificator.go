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

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	notificationStorage "github.com/Peripli/service-manager/notifications/postgres/storage"
	"github.com/Peripli/service-manager/storage"

	"github.com/lib/pq"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/notifications"
	"github.com/Peripli/service-manager/pkg/web"
)

const (
	postgresChannel             = "notifications"
	invalidRevisionNumber int64 = -1
	aTrue                 int32 = 1
	aFalse                int32 = 0
)

type consumers map[string][]notifications.NotificationQueue

type Notificator struct {
	isConnected int32
	isListening int32

	queueSize int

	connectionMutex *sync.Mutex
	connection      notificationStorage.NotificationConnection

	consumersMutex *sync.Mutex
	consumers      consumers
	storage        notificationStorage.NotificationStorage

	ctx context.Context

	lastKnownRevision int64
}

// NewNotificator returns new Notificator based on a given NotificatorStorage and desired queue size
func NewNotificator(st storage.Storage, storageSettings *storage.Settings, settings *Settings) (*Notificator, error) {
	ns, err := notificationStorage.NewNotificationStorage(st, storageSettings.URI, settings.MinReconnectInterval, settings.MaxReconnectInterval)
	if err != nil {
		return nil, err
	}
	return &Notificator{
		queueSize:         settings.NotificationQueuesSize,
		connectionMutex:   &sync.Mutex{},
		consumersMutex:    &sync.Mutex{},
		consumers:         make(consumers),
		storage:           ns,
		lastKnownRevision: invalidRevisionNumber,
	}, nil
}

// Start starts the Notificator. It must not be called concurrently.
func (n *Notificator) Start(ctx context.Context, group *sync.WaitGroup) error {
	if n.ctx != nil {
		return errors.New("notificator already started")
	}
	n.ctx = ctx
	if err := n.openConnection(); err != nil {
		return fmt.Errorf("could not open connection to database %v", err)
	}
	startInWaitGroup(n.awaitTermination, group)
	return nil
}

func (n *Notificator) RegisterConsumer(userContext *web.UserContext) (notifications.NotificationQueue, int64, error) {
	platform := &types.Platform{}
	err := userContext.Data.Data(platform)
	if err != nil {
		return nil, invalidRevisionNumber, fmt.Errorf("could not get platform from user context %v", err)
	}
	if platform.ID == "" {
		return nil, invalidRevisionNumber, errors.New("platform ID not found in user context")
	}
	if atomic.LoadInt32(&n.isConnected) == aFalse {
		return nil, invalidRevisionNumber, errors.New("cannot register consumer - Notificator is not running")
	}
	if err = n.startListening(); err != nil {
		return nil, invalidRevisionNumber, fmt.Errorf("listen to %s channel failed %v", postgresChannel, err)
	}
	queue, err := notifications.NewNotificationQueue(n.queueSize)
	if err != nil {
		return nil, invalidRevisionNumber, err
	}

	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()
	n.consumers[platform.ID] = append(n.consumers[platform.ID], queue)
	return queue, n.lastKnownRevision, nil
}

func (n *Notificator) UnregisterConsumer(queue notifications.NotificationQueue) error {
	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()

	platformIDToDelete, consumerIndex := n.findConsumer(queue.ID())
	if consumerIndex == -1 {
		return nil
	}
	platformConsumers := n.consumers[platformIDToDelete]
	n.consumers[platformIDToDelete] = append(platformConsumers[:consumerIndex], platformConsumers[consumerIndex+1:]...)
	queue.Close()

	if len(n.consumers[platformIDToDelete]) == 0 {
		delete(n.consumers, platformIDToDelete)
	}
	if len(n.consumers) == 0 {
		return n.stopListening()
	}
	return nil
}

func (n *Notificator) findConsumer(id string) (string, int) {
	for platformID, platformConsumers := range n.consumers {
		for index, consumer := range platformConsumers {
			if consumer.ID() == id {
				return platformID, index
			}
		}
	}
	return "", -1
}

func (n *Notificator) closeAllConsumers() {
	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()

	allConsumers := n.consumers
	n.consumers = make(consumers)
	for _, platformConsumers := range allConsumers {
		for _, consumer := range platformConsumers {
			consumer.Close()
		}
	}
}

func (n *Notificator) setConnection(conn notificationStorage.NotificationConnection) {
	n.connectionMutex.Lock()
	defer n.connectionMutex.Unlock()
	n.connection = conn
}

func (n *Notificator) openConnection() error {
	connection := n.storage.NewConnection(func(isConnected bool, err error) {
		if isConnected {
			atomic.StoreInt32(&n.isConnected, aTrue)
		} else {
			atomic.StoreInt32(&n.isConnected, aFalse)
			log.C(n.ctx).WithError(err).Info("connection to db closed, closing all consumers")
			n.closeAllConsumers()
		}
	})
	n.setConnection(connection)
	return nil
}

type notifyEventPayload struct {
	PlatformID     string `json:"platform_id"`
	NotificationID string `json:"notification_id"`
	Revision       int64  `json:"revision"`
}

func (n *Notificator) processNotifications(notificationChannel <-chan *pq.Notification) {
	defer func() {
		atomic.StoreInt32(&n.isListening, aFalse)
	}()
	for pqNotification := range notificationChannel {
		if pqNotification == nil {
			continue
		}
		payload, err := getPayload(pqNotification.Extra)
		if err != nil {
			log.C(n.ctx).WithError(err).Error("could not unmarshal notification payload")
			n.closeAllConsumers() // Ensures no notifications are lost
		} else {
			if err = n.processNotificationPayload(payload); err != nil {
				log.C(n.ctx).WithError(err).Error("closing consumers")
				n.closeAllConsumers() // Ensures no notifications are lost
			}
		}
	}
}

func getPayload(data string) (*notifyEventPayload, error) {
	payload := &notifyEventPayload{}
	if err := json.Unmarshal([]byte(data), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (n *Notificator) processNotificationPayload(payload *notifyEventPayload) error {
	notificationPlatformID := payload.PlatformID
	notificationID := payload.NotificationID

	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()
	n.lastKnownRevision = payload.Revision

	recipients := n.getRecipients(notificationPlatformID)
	if len(recipients) == 0 {
		return nil
	}
	notification, err := n.storage.GetNotification(n.ctx, notificationID)
	if err != nil {
		return fmt.Errorf("notification %s could not be retrieved from the DB: %v", notificationID, err.Error())
	}
	for _, platformConsumers := range recipients {
		n.sendNotificationToPlatformConsumers(platformConsumers, notification)
	}
	return nil
}

func (n *Notificator) getRecipients(platformID string) consumers {
	if platformID == "" {
		return n.consumers
	}
	platformConsumers, found := n.consumers[platformID]
	if !found {
		return nil
	}
	return consumers{
		platformID: platformConsumers,
	}
}

func (n *Notificator) sendNotificationToPlatformConsumers(platformConsumers []notifications.NotificationQueue, notification *types.Notification) {
	for _, consumer := range platformConsumers {
		if err := consumer.Enqueue(notification); err != nil {
			log.C(n.ctx).WithError(err).Infof("consumer %s notification queue returned error %v", consumer.ID(), err)
			consumer.Close()
		}
	}
}

func startInWaitGroup(f func(), group *sync.WaitGroup) {
	group.Add(1)
	go func() {
		defer group.Done()
		f()
	}()
}

func (n *Notificator) awaitTermination() {
	<-n.ctx.Done()
	logger := log.C(n.ctx)
	logger.Info("context cancelled, stopping Notificator...")
	n.stopConnection()
}

func (n *Notificator) stopConnection() {
	err := n.stopListening()
	logger := log.C(n.ctx)
	if err != nil {
		logger.WithError(err).Info("could not unlisten notification channel")
	}
	n.connectionMutex.Lock()
	defer n.connectionMutex.Unlock()
	if err = n.connection.Close(); err != nil {
		logger.WithError(err).Info("could not close db connection")
	}
}

func (n *Notificator) stopListening() error {
	n.connectionMutex.Lock()
	defer n.connectionMutex.Unlock()
	if atomic.LoadInt32(&n.isListening) == aFalse {
		return nil
	}
	return n.connection.Unlisten(postgresChannel)
}

func (n *Notificator) startListening() error {
	n.connectionMutex.Lock()
	defer n.connectionMutex.Unlock()
	if atomic.LoadInt32(&n.isListening) == aTrue {
		return nil
	}
	err := n.connection.Listen(postgresChannel)
	if err != nil {
		return err
	}
	lastKnownRevision, err := n.storage.GetLastRevision(n.ctx)
	if err != nil {
		if errUnlisten := n.connection.Unlisten(postgresChannel); errUnlisten != nil {
			log.C(n.ctx).WithError(errUnlisten).Errorf("could not unlisten %s channel", postgresChannel)
		}
		return err
	}
	n.lastKnownRevision = lastKnownRevision
	atomic.StoreInt32(&n.isListening, aTrue)
	go n.processNotifications(n.connection.NotificationChannel())
	return nil
}
