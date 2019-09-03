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
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	notificationConnection "github.com/Peripli/service-manager/storage/postgres/notification_connection"

	"github.com/lib/pq"
)

const (
	postgresChannel       = "notifications"
	dbPingInterval        = time.Second * 60
	aTrue           int32 = 1
	aFalse          int32 = 0
)

type Notificator struct {
	isListening bool // To be used only under connectionMutex.Lock
	isConnected int32

	queueSize int

	connectionMutex *sync.Mutex
	connection      notificationConnection.NotificationConnection

	consumersMutex    *sync.Mutex
	consumers         *consumers
	storage           notificationStorage
	connectionCreator notificationConnectionCreator

	// stopProcessing is used to cancel the go routine which processes notifications.
	// It closes all consumers and stops listening to the postgres notification channel.
	stopProcessing      context.CancelFunc
	notificationFilters []storage.ReceiversFilterFunc
	ctx                 context.Context

	lastKnownRevision int64
	dbPingInterval    time.Duration
}

// NewNotificator returns new Notificator based on a given NotificatorStorage and desired queue size
func NewNotificator(st storage.Storage, settings *storage.Settings) (*Notificator, error) {
	ns, err := NewNotificationStorage(st)
	connectionCreator := &notificationConnectionCreatorImpl{
		storageURI:           settings.URI,
		minReconnectInterval: settings.Notification.MinReconnectInterval,
		maxReconnectInterval: settings.Notification.MaxReconnectInterval,
	}
	if err != nil {
		return nil, err
	}

	return &Notificator{
		queueSize:       settings.Notification.QueuesSize,
		connectionMutex: &sync.Mutex{},
		consumersMutex:  &sync.Mutex{},
		consumers: &consumers{
			queues:    make(map[string][]storage.NotificationQueue),
			platforms: make([]*types.Platform, 0),
		},
		storage:           ns,
		connectionCreator: connectionCreator,
		stopProcessing:    func() {},
		lastKnownRevision: types.InvalidRevision,
		dbPingInterval:    dbPingInterval,
	}, nil
}

// Start starts the Notificator. It must not be called concurrently.
func (n *Notificator) Start(ctx context.Context, group *sync.WaitGroup) error {
	if n.ctx != nil {
		return errors.New("notificator already started")
	}
	n.ctx = ctx
	n.setConnection(n.connectionCreator.NewConnection(func(isConnected bool, err error) {
		if isConnected {
			atomic.StoreInt32(&n.isConnected, aTrue)
			log.C(n.ctx).Info("DB connection for notifications established")
		} else {
			atomic.StoreInt32(&n.isConnected, aFalse)
			log.C(n.ctx).WithError(err).Error("DB connection for notifications closed")
			n.stopProcessing() // closes all consumers and stops processing notifications
		}
	}))
	util.StartInWaitGroupWithContext(ctx, func(c context.Context) {
		<-c.Done()
		log.C(c).Info("context cancelled, stopping Notificator...")
		n.stopConnection()
	}, group)
	return nil
}

func (n *Notificator) addConsumer(platform *types.Platform, queue storage.NotificationQueue) (int64, error) {
	// must listen and add consumer under connectionMutex lock as UnregisterConsumer
	// might stop notification processing if no other consumers are present
	n.connectionMutex.Lock()
	defer n.connectionMutex.Unlock()
	if !n.isListening {
		log.C(n.ctx).Debugf("Start listening notification channel %s", postgresChannel)
		err := n.connection.Listen(postgresChannel)
		if err != nil && err != pq.ErrChannelAlreadyOpen {
			return types.InvalidRevision, fmt.Errorf("listen to %s channel failed %v", postgresChannel, err)
		}
		lastKnownRevision, err := n.storage.GetLastRevision(n.ctx)
		if err != nil {
			if errUnlisten := n.connection.Unlisten(postgresChannel); errUnlisten != nil {
				log.C(n.ctx).WithError(errUnlisten).Errorf("could not unlisten %s channel", postgresChannel)
			}
			return types.InvalidRevision, fmt.Errorf("getting last revision failed %v", err)
		}
		atomic.StoreInt64(&n.lastKnownRevision, lastKnownRevision)
		n.isListening = true
		notificationProcessingContext, stopProcessing := context.WithCancel(n.ctx)
		n.stopProcessing = stopProcessing
		go n.processNotifications(n.connection.NotificationChannel(), notificationProcessingContext)
	} else {
		log.C(n.ctx).Debugf("Already listening to notification channel %s", postgresChannel)
	}
	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()
	n.consumers.Add(platform, queue)
	return atomic.LoadInt64(&n.lastKnownRevision), nil
}

func (n *Notificator) RegisterConsumer(consumer *types.Platform, lastKnownRevision int64) (storage.NotificationQueue, int64, error) {
	if atomic.LoadInt32(&n.isConnected) == aFalse {
		return nil, types.InvalidRevision, errors.New("cannot register consumer - Notificator is not running")
	}
	queue, err := storage.NewNotificationQueue(n.queueSize)
	if err != nil {
		return nil, types.InvalidRevision, err
	}

	var lastKnownRevisionToSM int64
	lastKnownRevisionToSM, err = n.addConsumer(consumer, queue)
	if err != nil {
		return nil, types.InvalidRevision, err
	}
	if lastKnownRevision == types.InvalidRevision || lastKnownRevision == lastKnownRevisionToSM {
		return queue, lastKnownRevisionToSM, nil
	}
	defer func() {
		if err != nil {
			if errUnregisterConsumer := n.UnregisterConsumer(queue); errUnregisterConsumer != nil {
				log.C(n.ctx).WithError(errUnregisterConsumer).Errorf("Could not unregister notification consumer %s", queue.ID())
			}
		}
	}()
	if lastKnownRevision > lastKnownRevisionToSM {
		log.C(n.ctx).Debug("lastKnownRevision is grater than the one SM knows")
		err = util.ErrInvalidNotificationRevision // important for defer logic
		return nil, types.InvalidRevision, err
	}
	var queueWithMissedNotifications storage.NotificationQueue
	queueWithMissedNotifications, err = n.replaceQueueWithMissingNotificationsQueue(queue, lastKnownRevision, lastKnownRevisionToSM, consumer)
	if err != nil {
		return nil, types.InvalidRevision, err
	}
	return queueWithMissedNotifications, lastKnownRevisionToSM, nil
}

func (n *Notificator) filterRecipients(recipients []*types.Platform, notification *types.Notification) []*types.Platform {
	for _, filter := range n.notificationFilters {
		recipients = filter(recipients, notification)
		if len(recipients) == 0 {
			return recipients
		}
	}
	return recipients
}

func (n *Notificator) replaceQueueWithMissingNotificationsQueue(queue storage.NotificationQueue, lastKnownRevision, lastKnownRevisionToSM int64, platform *types.Platform) (storage.NotificationQueue, error) {
	if _, err := n.storage.GetNotificationByRevision(n.ctx, lastKnownRevision); err != nil {
		if err == util.ErrNotFoundInStorage {
			log.C(n.ctx).WithError(err).Debugf("Notification with revision %d not found in storage", lastKnownRevision)
			return nil, util.ErrInvalidNotificationRevision
		}
		return nil, err
	}

	missedNotifications, err := n.storage.ListNotifications(n.ctx, platform.ID, lastKnownRevision, lastKnownRevisionToSM)
	if err != nil {
		return nil, err
	}
	filteredMissedNotification := make([]*types.Notification, 0, len(missedNotifications))
	for _, notification := range missedNotifications {
		recipients := n.filterRecipients([]*types.Platform{platform}, notification)
		if len(recipients) != 0 {
			filteredMissedNotification = append(filteredMissedNotification, notification)
		}
	}

	if n.queueSize < len(filteredMissedNotification) {
		log.C(n.ctx).Debugf("Too many missed notifications %d", len(filteredMissedNotification))
		return nil, util.ErrInvalidNotificationRevision
	}

	queueWithMissedNotifications, err := storage.NewNotificationQueue(n.queueSize)
	if err != nil {
		return nil, err
	}
	for _, notification := range filteredMissedNotification {
		if err = queueWithMissedNotifications.Enqueue(notification); err != nil {
			return nil, err
		}
	}

	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()
	for {
		select {
		case notification, ok := <-queue.Channel():
			if !ok {
				return nil, errors.New("notification queue has been closed")
			}
			if err = queueWithMissedNotifications.Enqueue(notification); err != nil {
				return nil, err
			}
		default:
			if err = n.consumers.ReplaceQueue(queue.ID(), queueWithMissedNotifications); err != nil {
				return nil, err
			}
			return queueWithMissedNotifications, nil
		}
	}
}

func (n *Notificator) UnregisterConsumer(queue storage.NotificationQueue) error {
	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()
	queue.Close()
	if n.consumers.Len() == 0 {
		return nil // Consumer already unregistered
	}
	n.consumers.Delete(queue)
	if n.consumers.Len() == 0 {
		log.C(n.ctx).Debugf("No notification consumers left. Stop listening to channel %s", postgresChannel)
		n.stopProcessing() // stop processing notifications as there are no consumers
	}
	return nil
}

// RegisterFilter adds new notification filter. It must not be called concurrently.
func (n *Notificator) RegisterFilter(f storage.ReceiversFilterFunc) {
	n.notificationFilters = append(n.notificationFilters, f)
}

func (n *Notificator) closeAllConsumers() {
	n.consumersMutex.Lock()
	defer n.consumersMutex.Unlock()

	platformConsumers := n.consumers.Clear()
	for _, platformConsumers := range platformConsumers {
		for _, queue := range platformConsumers {
			queue.Close()
		}
	}
}

func (n *Notificator) setConnection(conn notificationConnection.NotificationConnection) {
	n.connectionMutex.Lock()
	defer n.connectionMutex.Unlock()
	n.connection = conn
}

type notifyEventPayload struct {
	PlatformID     string `json:"platform_id"`
	NotificationID string `json:"notification_id"`
	Revision       int64  `json:"revision"`
}

func (n *Notificator) processNotifications(notificationChannel <-chan *pq.Notification, processingContext context.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.C(n.ctx).Errorf("recovered from panic while processing notifications: %s", err)
		}
	}()
	defer func() {
		n.connectionMutex.Lock()
		defer n.connectionMutex.Unlock()
		n.isListening = false
		n.stopProcessing() // closing processingContext if not already closed
		n.closeAllConsumers()
		log.C(n.ctx).Debugf("Stop listening notification channel %s", postgresChannel)
		if atomic.LoadInt32(&n.isConnected) == aTrue {
			if err := n.connection.Unlisten(postgresChannel); err != nil {
				log.C(n.ctx).WithError(err).Errorf("Could not unlisten channel %s", postgresChannel)
			}
		}
	}()
	lastNotificationReceived := time.Now()
	for {
		select {
		case pqNotification, ok := <-notificationChannel:
			if !ok {
				log.C(n.ctx).Error("Notification channel closed")
				return
			}
			if pqNotification == nil { // when connection is re-established a nil notification is sent by the library
				log.C(n.ctx).Debug("Empty notification received")
				continue
			}
			lastNotificationReceived = time.Now()
			log.C(n.ctx).Debugf("Received new notification from channel %s", pqNotification.Channel)
			payload, err := getPayload(pqNotification.Extra)
			if err != nil {
				log.C(n.ctx).WithError(err).Error("Could not unmarshal notification payload. Closing consumers...")
				return
			} else {
				if err = n.processNotificationPayload(payload); err != nil {
					log.C(n.ctx).WithError(err).Error("Could not process notification payload. Closing consumers...")
					return
				}
			}
		case <-time.After(n.dbPingInterval):
			log.C(n.ctx).Debugf("No notifications in %s. Pinging connection", time.Since(lastNotificationReceived))
			if err := n.connection.Ping(); err != nil {
				log.C(n.ctx).WithError(err).Error("Pinging connection failed. Closing all consumers...")
				return
			}
		case <-processingContext.Done():
			log.C(n.ctx).Debug("Stopping processing of notifications. Closing consumers...")
			return
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
	atomic.StoreInt64(&n.lastKnownRevision, payload.Revision)

	recipients := n.getRecipients(notificationPlatformID)
	if len(recipients) == 0 {
		log.C(n.ctx).Debugf("No recipients to receive notification %s", notificationID)
		return nil
	}
	notification, err := n.storage.GetNotification(n.ctx, notificationID)
	if err != nil {
		return fmt.Errorf("notification %s could not be retrieved from the DB: %v", notificationID, err.Error())
	}
	recipients = n.filterRecipients(recipients, notification)
	log.C(n.ctx).Debugf("%d platforms should receive notification %s", len(recipients), notificationID)
	for _, platform := range recipients {
		platformID := platform.ID
		n.sendNotificationToPlatformConsumers(platformID, n.consumers.GetQueuesForPlatform(platformID), notification)
	}
	return nil
}

func (n *Notificator) getRecipients(platformID string) []*types.Platform {
	if platformID == "" {
		return n.consumers.platforms
	}
	platform := n.consumers.GetPlatform(platformID)
	if platform == nil {
		return nil
	}
	return []*types.Platform{platform}
}

func (n *Notificator) sendNotificationToPlatformConsumers(platformID string, platformConsumers []storage.NotificationQueue, notification *types.Notification) {
	log.C(n.ctx).Debugf("Sending notification %s to %d consumers for platform %s", notification.ID, len(platformConsumers), platformID)
	for _, consumer := range platformConsumers {
		if err := consumer.Enqueue(notification); err != nil {
			log.C(n.ctx).WithError(err).Infof("Consumer %s notification queue returned error %v", consumer.ID(), err)
			consumer.Close()
		}
	}
}

func (n *Notificator) stopConnection() {
	n.stopProcessing() // stop processing notifications
	n.connectionMutex.Lock()
	defer n.connectionMutex.Unlock()

	atomic.StoreInt32(&n.isConnected, aFalse)
	if err := n.connection.Close(); err != nil {
		log.C(n.ctx).WithError(err).Error("Could not close db connection")
	}
}

type consumers struct {
	queues    map[string][]storage.NotificationQueue
	platforms []*types.Platform
}

func (c *consumers) find(queueID string) (string, int) {
	for platformID, notificationQueues := range c.queues {
		for index, queue := range notificationQueues {
			if queue.ID() == queueID {
				return platformID, index
			}
		}
	}
	return "", -1
}

func (c *consumers) ReplaceQueue(queueID string, newQueue storage.NotificationQueue) error {
	platformID, queueIndex := c.find(queueID)
	if queueIndex == -1 {
		return fmt.Errorf("could not find consumer with id %s", queueID)
	}
	c.queues[platformID][queueIndex] = newQueue
	return nil
}

func (c *consumers) Delete(queue storage.NotificationQueue) {
	platformIDToDelete, queueIndex := c.find(queue.ID())
	if queueIndex == -1 {
		return
	}
	platformConsumers := c.queues[platformIDToDelete]
	c.queues[platformIDToDelete] = append(platformConsumers[:queueIndex], platformConsumers[queueIndex+1:]...)

	if len(c.queues[platformIDToDelete]) == 0 {
		delete(c.queues, platformIDToDelete)
		for index, platform := range c.platforms {
			if platform.ID == platformIDToDelete {
				c.platforms = append(c.platforms[:index], c.platforms[index+1:]...)
				break
			}
		}
	}
}

func (c *consumers) Add(platform *types.Platform, queue storage.NotificationQueue) {
	if len(c.queues[platform.ID]) == 0 {
		c.platforms = append(c.platforms, platform)
	}
	c.queues[platform.ID] = append(c.queues[platform.ID], queue)
}

func (c *consumers) Clear() map[string][]storage.NotificationQueue {
	allQueues := c.queues
	c.queues = make(map[string][]storage.NotificationQueue)
	c.platforms = make([]*types.Platform, 0)
	return allQueues
}

func (c *consumers) Len() int {
	return len(c.queues)
}

func (c *consumers) GetPlatform(platformID string) *types.Platform {
	for _, platform := range c.platforms {
		if platform.ID == platformID {
			return platform
		}
	}
	return nil
}

func (c *consumers) GetQueuesForPlatform(platformID string) []storage.NotificationQueue {
	return c.queues[platformID]
}
