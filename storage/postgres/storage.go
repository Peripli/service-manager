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
	"fmt"
	"sync"
	"time"

	"github.com/Peripli/service-manager/storage"
	"github.com/golang-migrate/migrate"
	migratepg "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
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
	mutex         *sync.Mutex
}

func (storage *postgresStorage) checkOpen() {
	if storage.db == nil {
		logrus.Panicln("Storage is not yet Open")
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
	if storage.db == nil {
		logrus.Panicln("Storage is not yet Open")
	}
	return &credentialStorage{storage.db}
}

func (storage *postgresStorage) Security() storage.Security {
	storage.checkOpen()
	return &securityStorage{storage.db, storage.encryptionKey, false, storage.mutex}
}

func (storage *postgresStorage) Open(uri string, encryptionKey []byte) error {
	var err error
	if uri == "" {
		return fmt.Errorf("storage URI cannot be empty")
	}
	if storage.db == nil {
		storage.db, err = sqlx.Connect(Storage, uri)
		if err != nil {
			logrus.Panicln("Could not connect to PostgreSQL:", err)
		}
		storage.state = &storageState{
			storageError:         nil,
			lastCheck:            time.Now(),
			mutex:                &sync.RWMutex{},
			db:                   storage.db,
			storageCheckInterval: time.Second * 5,
		}
		storage.mutex = &sync.Mutex{}
		storage.encryptionKey = encryptionKey
		logrus.Debug("Updating database schema")
		if err := updateSchema(storage.db); err != nil {
			logrus.Panicln("Could not update database schema:", err)
		}
	}
	return err
}

func (storage *postgresStorage) Close() error {
	storage.checkOpen()
	return storage.db.Close()
}

func updateSchema(db *sqlx.DB) error {
	driver, err := migratepg.WithInstance(db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file://storage/postgres/migrations", "postgres", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err == migrate.ErrNoChange {
		logrus.Debug("Database schema already up to date")
		err = nil
	}
	return err
}
