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

// Package storage contains logic around the Service Manager persistent storage
package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path"
	"runtime"
	"sync"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

type Entity interface {
	GetID() string
	ToObject() (types.Object, error)
	FromObject(object types.Object) (Entity, error)
	NewLabel(id, entityID, key, value string) Label
}

type Label interface {
	GetKey() string
	GetValue() string
}

type EntityMetadata struct {
	Name      string
	TableName string
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = path.Dir(b)
)

// Settings type to be loaded from the environment
type Settings struct {
	URI                string                `mapstructure:"uri" description:"URI of the storage"`
	MigrationsURL      string                `mapstructure:"migrations_url" description:"location of a directory containing sql migrations scripts"`
	EncryptionKey      string                `mapstructure:"encryption_key" description:"key to use for encrypting database entries"`
	SkipSSLValidation  bool                  `mapstructure:"skip_ssl_validation" description:"whether to skip ssl verification when connecting to the storage"`
	SSLMode            string                `mapstructure:"sslmode" description:"defines ssl mode type"`
	SSLRootCert        string                `mapstructure:"sslrootcert" description:"The location of the root certificate file."`
	MaxIdleConnections int                   `mapstructure:"max_idle_connections" description:"sets the maximum number of connections in the idle connection pool"`
	MaxOpenConnections int                   `mapstructure:"max_open_connections" description:"sets the maximum number of open connections to the database"`
	ReadTimeout        int                   `mapstructure:"read_timeout" description:"sets the limit for reading in milliseconds"`
	WriteTimeout       int                   `mapstructure:"write_timeout" description:"sets the limit for writing in milliseconds"`
	Notification       *NotificationSettings `mapstructure:"notification"`
	IntegrityProcessor security.IntegrityProcessor
}

// DefaultSettings returns default values for storage settings
func DefaultSettings() *Settings {
	return &Settings{
		URI:                "",
		MigrationsURL:      fmt.Sprintf("file://%s/postgres/migrations", basepath),
		EncryptionKey:      "",
		SkipSSLValidation:  false,
		MaxIdleConnections: 5,
		MaxOpenConnections: 30,
		ReadTimeout:        900000, //15 minutes
		WriteTimeout:       900000, //15 minutes
		Notification:       DefaultNotificationSettings(),
		IntegrityProcessor: &security.HashingIntegrityProcessor{
			HashingFunc: func(data []byte) []byte {
				hash := sha256.Sum256(data)
				return hash[:]
			},
		},
	}
}

// Validate validates the storage settings
func (s *Settings) Validate() error {
	if len(s.URI) == 0 {
		return fmt.Errorf("validate Settings: StorageURI missing")
	}
	if len(s.MigrationsURL) == 0 {
		return fmt.Errorf("validate Settings: StorageMigrationsURL missing")
	}
	if len(s.EncryptionKey) != 32 {
		return fmt.Errorf("validate Settings: StorageEncryptionKey must be exactly 32 symbols long but was %d symbols long", len(s.EncryptionKey))
	}
	if s.IntegrityProcessor == nil {
		return fmt.Errorf("validate Settings: StorageIntegrityProcessor must not be nil")
	}
	return s.Notification.Validate()
}

// NotificationSettings type to be loaded from the environment
type NotificationSettings struct {
	QueuesSize           int           `mapstructure:"queues_size" description:"maximum number of notifications queued for sending to a client"`
	MinReconnectInterval time.Duration `mapstructure:"min_reconnect_interval" description:"minimum timeout between storage listen reconnects"`
	MaxReconnectInterval time.Duration `mapstructure:"max_reconnect_interval" description:"maximum timeout between storage listen reconnects"`
	CleanInterval        time.Duration `mapstructure:"clean_interval" description:"time between notification clean-up"`
	KeepFor              time.Duration `mapstructure:"keep_for" description:"the time to keep a notification in the storage"`
}

// DefaultNotificationSettings returns default values for Notificator settings
func DefaultNotificationSettings() *NotificationSettings {
	return &NotificationSettings{
		QueuesSize:           100,
		MinReconnectInterval: time.Millisecond * 200,
		MaxReconnectInterval: time.Second * 20,
		CleanInterval:        time.Minute * 15,
		KeepFor:              time.Hour * 12,
	}
}

// Validate validates the Notification settings
func (s *NotificationSettings) Validate() error {
	if s.QueuesSize < 1 {
		return fmt.Errorf("notification queues size (%d) should be at lest 1", s.QueuesSize)
	}
	if s.MinReconnectInterval > s.MaxReconnectInterval {
		return fmt.Errorf("min reconnect interval (%s) should not be greater than max reconnect interval (%s)",
			s.MinReconnectInterval, s.MaxReconnectInterval)
	}
	if s.MinReconnectInterval < 0 {
		return fmt.Errorf("notification minimum reconnect interval (%d) should be grater or equal to 0", s.MinReconnectInterval)
	}
	if s.KeepFor < 0 {
		return fmt.Errorf("notification keep for (%d) should be grater or equal to 0", s.KeepFor)
	}
	if s.CleanInterval < 0 {
		return fmt.Errorf("notification clean interval (%d) should be grater or equal to 0", s.CleanInterval)
	}
	return nil
}

// OpenCloser represents an openable and closeable storage
type OpenCloser interface {
	// Open initializes the storage, e.g. opens a connection to the underlying storage
	Open(options *Settings) error

	// Close clears resources associated with this storage, e.g. closes the connection the underlying storage
	Close() error
}

// Pinger allows pinging the storage to check liveliness
//go:generate counterfeiter . Pinger
type Pinger interface {
	// PingContext verifies a connection to the database is still alive, establishing a connection if necessary.
	PingContext(context.Context) error
}

// PingFunc is an adapter that allows to use regular functions as Pinger
type PingFunc func(context.Context) error

// PingContext allows PingFunc to act as a Pinger
func (mf PingFunc) PingContext(ctx context.Context) error {
	return mf(ctx)
}

type Repository interface {
	// Create stores an object in SM DB
	Create(ctx context.Context, obj types.Object) (types.Object, error)

	// Get retrieves an object using the provided id from SM DB
	Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error)

	// GetForUpdate retrieves an object using the provided id from SM DB while also locking the retrieved rows
	GetForUpdate(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error)

	// List retrieves all object from SM DB
	List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error)

	// ListNoLabels retrieves all object from SM DB without their labels
	ListNoLabels(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error)

	// Count retrieves number of objects of particular type in SM DB
	Count(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error)

	// Count label values of retrieved objects of particular type in SM DB
	CountLabelValues(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error)

	// Query for list retrieves a list of items using a named query
	QueryForList(ctx context.Context, objectType types.ObjectType, queryName NamedQuery, queryParams map[string]interface{}) (types.ObjectList, error)

	// DeleteReturning deletes objects from SM DB
	DeleteReturning(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error)

	//Delete deletes objects from SM DB
	Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) error

	// Update updates an object from SM DB
	Update(ctx context.Context, obj types.Object, labelChanges types.LabelChanges, criteria ...query.Criterion) (types.Object, error)

	// UpdateLabels updates the object labels in SM DB
	UpdateLabels(ctx context.Context, objectType types.ObjectType, objectID string, labelChanges types.LabelChanges, _ ...query.Criterion) error

	// Retrieves all the registered entities
	GetEntities() []EntityMetadata
}

// TransactionalRepository is a storage repository that can initiate a transaction
type TransactionalRepository interface {
	Repository

	// InTransaction initiates a transaction and allows passing a function to be executed within the transaction
	InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error
}

// TransactionalRepositoryDecorator allows decorating a TransactionalRepository
type TransactionalRepositoryDecorator func(TransactionalRepository) (TransactionalRepository, error)

// Storage interface provides entity-specific storages
//go:generate counterfeiter . Storage
type Storage interface {
	OpenCloser
	TransactionalRepository
	Pinger

	Introduce(entity Entity)
}

// ErrQueueClosed error stating that the queue is closed
var ErrQueueClosed = errors.New("queue closed")

// ErrQueueFull error stating that the queue is full
var ErrQueueFull = errors.New("queue is full")

// NotificationQueue is used for receiving notifications
//go:generate counterfeiter . NotificationQueue
type NotificationQueue interface {
	// Enqueue adds a new notification for processing.
	Enqueue(notification *types.Notification) error

	// Channel returns the go channel with received notifications which has to be processed.
	Channel() <-chan *types.Notification

	// Close closes the queue.
	Close()

	// ID returns unique queue identifier
	ID() string
}

// Notificator is used for receiving notifications for SM events
//go:generate counterfeiter . Notificator
type Notificator interface {
	// Start starts the Notificator
	Start(ctx context.Context, group *sync.WaitGroup) error

	// RegisterConsumer returns notification queue, last_known_revision and error if any.
	// Notifications after lastKnownRevision will be added to the queue.
	// If lastKnownRevision is -1 no previous notifications will be sent.
	// When consumer wants to stop listening for notifications it must unregister the notification queue.
	RegisterConsumer(consumer *types.Platform, lastKnownRevision int64) (NotificationQueue, int64, error)

	// UnregisterConsumer must be called to stop receiving notifications in the queue
	UnregisterConsumer(queue NotificationQueue) error

	// RegisterFilter adds a new filter which decides if a platform should receive given notification
	RegisterFilter(f ReceiversFilterFunc)
}

// ReceiversFilterFunc filters recipients for a given notifications
type ReceiversFilterFunc func(recipients []*types.Platform, notification *types.Notification) (filteredRecipients []*types.Platform)
