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
	"fmt"
	"path"
	"runtime"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
)

type Entity interface {
	GetID() string
	ToObject() types.Object
	FromObject(object types.Object) (Entity, bool)
	BuildLabels(labels types.Labels, newLabel func(id, key, value string) Label) ([]Label, error)
	NewLabel(id, key, value string) Label
}

type Label interface {
	GetKey() string
	GetValue() string
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = path.Dir(b)
)

// Settings type to be loaded from the environment
type Settings struct {
	URI                string `mapstructure:"uri" description:"URI of the storage"`
	MigrationsURL      string `mapstructure:"migrations_url" description:"location of a directory containing sql migrations scripts"`
	EncryptionKey      string `mapstructure:"encryption_key" description:"key to use for encrypting database entries"`
	SkipSSLValidation  bool   `mapstructure:"skip_ssl_validation" description:"whether to skip ssl verification when connecting to the storage"`
	MaxIdleConnections int    `mapstructure:"max_idle_connections" description:"sets the maximum number of connections in the idle connection pool"`
}

// DefaultSettings returns default values for storage settings
func DefaultSettings() *Settings {
	return &Settings{
		URI:                "",
		MigrationsURL:      fmt.Sprintf("file://%s/postgres/migrations", basepath),
		EncryptionKey:      "",
		SkipSSLValidation:  false,
		MaxIdleConnections: 5,
	}
}

// Validate validates the storage settings
func (s *Settings) Validate() error {
	if len(s.URI) == 0 {
		return fmt.Errorf("validate Settings: StorageURI missing")
	}
	if len(s.EncryptionKey) != 32 {
		return fmt.Errorf("validate Settings: StorageEncryptionKey must be exactly 32 symbols long but was %d symbols long", len(s.EncryptionKey))
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
	// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
	Ping() error
}

// PingFunc is an adapter that allows to use regular functions as Pinger
type PingFunc func() error

// Ping allows PingFunc to act as a Pinger
func (mf PingFunc) Ping() error {
	return mf()
}

type Repository interface {
	// Create stores a broker in SM DB
	Create(ctx context.Context, obj types.Object) (string, error)

	// Get retrieves a broker using the provided id from SM DB
	Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error)

	// List retrieves all brokers from SM DB
	List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error)

	// Delete deletes a broker from SM DB
	Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error)

	// Update updates a broker from SM DB
	Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)

	Credentials() Credentials
	Security() Security
}

// TransactionalRepository is a storage repository that can initiate a transaction
type TransactionalRepository interface {
	Repository

	// InTransaction initiates a transaction and allows passing a function to be executed within the transaction
	InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error
}

// Storage interface provides entity-specific storages
//go:generate counterfeiter . Storage
type Storage interface {
	OpenCloser
	Pinger
	TransactionalRepository

	Introduce(entity Entity)
}

// Credentials interface for Credentials db operations
//go:generate counterfeiter . Credentials
type Credentials interface {
	// Get retrieves credentials using the provided username from SM DB
	Get(ctx context.Context, username string) (*types.Credentials, error)
}

// Security interface for encryption key operations
type Security interface {
	// Lock locks the storage so that only one process can manipulate the encryption key.
	// Returns an error if the process has already acquired the lock
	Lock(ctx context.Context) error

	// Unlock releases the acquired lock.
	Unlock(ctx context.Context) error

	// Fetcher provides means to obtain the encryption key
	Fetcher() security.KeyFetcher

	// Setter provides means to change the encryption  key
	Setter() security.KeySetter
}
