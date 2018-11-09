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

// Package postgres implements the Service Manager storage interfaces for Postgresql Storage
package postgres

import (
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"
	"github.com/golang-migrate/migrate"
	migratepg "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
)

// Storage defines the name of the PostgreSQL relational storage
const Storage = "postgres"

func init() {
	storage.Register(Storage, &postgresStorage{})
}

type postgresStorage struct {
	db            *sqlx.DB
	state         *storageState
	encryptionKey []byte
}

func (storage *postgresStorage) checkOpen() {
	if storage.db == nil {
		log.D().Panicln("Storage is not yet Open")
	}
}

func (storage *postgresStorage) Ping() error {
	storage.checkOpen()
	return storage.state.Get()
}

func (storage *postgresStorage) Broker() storage.Broker {
	storage.checkOpen()
	return &brokerStorage{storage.db}
}

func (storage *postgresStorage) Platform() storage.Platform {
	storage.checkOpen()
	return &platformStorage{storage.db}
}

func (storage *postgresStorage) Credentials() storage.Credentials {
	storage.checkOpen()
	return &credentialStorage{storage.db}
}

func (storage *postgresStorage) Security() storage.Security {
	storage.checkOpen()
	return &securityStorage{storage.db, storage.encryptionKey, false, &sync.Mutex{}}
}

func (storage *postgresStorage) Open(options *storage.Settings) error {
	var err error
	if err = options.Validate(); err != nil {
		return err
	}
	if storage.db == nil {
		storage.db, err = sqlx.Connect(Storage, options.URI)
		if err != nil {
			log.D().Panicln("Could not connect to PostgreSQL:", err)
		}
		storage.state = &storageState{
			lastCheckTime:        time.Now(),
			mutex:                &sync.RWMutex{},
			db:                   storage.db,
			storageCheckInterval: time.Second * 5,
		}
		storage.encryptionKey = []byte(options.EncryptionKey)
		log.D().Debug("Updating database schema")
		if err := storage.updateSchema(options.MigrationsURL); err != nil {
			log.D().Panicln("Could not update database schema:", err)
		}
	}
	return err
}

func (storage *postgresStorage) Close() error {
	storage.checkOpen()
	return storage.db.Close()
}

func (storage *postgresStorage) updateSchema(migrationsURL string) error {
	driver, err := migratepg.WithInstance(storage.db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsURL, "postgres", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err == migrate.ErrNoChange {
		log.D().Debug("Database schema already up to date")
		err = nil
	}
	return err
}
