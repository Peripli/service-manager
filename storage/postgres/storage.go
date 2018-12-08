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

// Package postgres implements the Service Manager storage interfaces for Postgresql Repository
package postgres

import (
	"context"
	"fmt"
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

type transactionalWarehouse struct {
	tx *sqlx.Tx
}

func (ts *transactionalWarehouse) ServiceOffering() storage.ServiceOffering {
	ts.checkOpen()
	return &serviceOfferingStorage{db: ts.tx}
}

func (ts *transactionalWarehouse) ServicePlan() storage.ServicePlan {
	ts.checkOpen()
	return &servicePlanStorage{db: ts.tx}
}

func (ts *transactionalWarehouse) Visibility() storage.Visibility {
	ts.checkOpen()
	return &visibilityStorage{db: ts.tx}
}

func (ts *transactionalWarehouse) Security() storage.Security {
	ts.checkOpen()
	return &securityStorage{db: ts.tx}
}

func (ts *transactionalWarehouse) Broker() storage.Broker {
	ts.checkOpen()
	return &brokerStorage{db: ts.tx}
}

func (ts *transactionalWarehouse) Platform() storage.Platform {
	ts.checkOpen()
	return &platformStorage{db: ts.tx}
}

func (ts *transactionalWarehouse) Credentials() storage.Credentials {
	ts.checkOpen()
	return &credentialStorage{db: ts.tx}
}

func (ts *transactionalWarehouse) checkOpen() {
	if ts.tx == nil {
		log.D().Panicln("Storage transaction is not present for transactional warehouse")
	}
}

func (ps *postgresStorage) InTransaction(ctx context.Context, f func(ctx context.Context, transactionalStorage storage.Warehouse) error) error {
	ok := false
	tx, err := ps.db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if !ok {
			if txError := tx.Rollback(); txError != nil {
				log.C(ctx).Error("Could not rollback transaction", txError)
			}
		}
	}()

	transactionalStorage := &transactionalWarehouse{
		tx: tx,
	}

	if err := f(ctx, transactionalStorage); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	ok = true
	return nil
}

func (ps *postgresStorage) Ping() error {
	ps.checkOpen()
	return ps.state.Get()
}

func (ps *postgresStorage) Broker() storage.Broker {
	ps.checkOpen()
	return &brokerStorage{ps.db}
}

func (ps *postgresStorage) Platform() storage.Platform {
	ps.checkOpen()
	return &platformStorage{ps.db}
}

func (ps *postgresStorage) Credentials() storage.Credentials {
	ps.checkOpen()
	return &credentialStorage{ps.db}
}

func (ps *postgresStorage) ServiceOffering() storage.ServiceOffering {
	return &serviceOfferingStorage{ps.db}
}

func (ps *postgresStorage) ServicePlan() storage.ServicePlan {
	return &servicePlanStorage{ps.db}
}

func (ps *postgresStorage) Visibility() storage.Visibility {
	return &visibilityStorage{ps.db}
}

func (ps *postgresStorage) Security() storage.Security {
	ps.checkOpen()
	return &securityStorage{ps.db, ps.encryptionKey, false, &sync.Mutex{}}
}

func (ps *postgresStorage) Open(options *storage.Settings) error {
	var err error
	if err = options.Validate(); err != nil {
		return err
	}
	if len(options.MigrationsURL) == 0 {
		return fmt.Errorf("validate Settings: StorageMigrationsURL missing")
	}
	if ps.db == nil {
		sslModeParam := ""
		if options.SkipSSLValidation {
			sslModeParam = "?sslmode=disable"
		}
		ps.db, err = sqlx.Connect(Storage, options.URI+sslModeParam)
		if err != nil {
			log.D().Panicln("Could not connect to PostgreSQL:", err)
		}
		ps.state = &storageState{
			lastCheckTime:        time.Now(),
			mutex:                &sync.RWMutex{},
			db:                   ps.db,
			storageCheckInterval: time.Second * 5,
		}
		ps.encryptionKey = []byte(options.EncryptionKey)
		log.D().Debugf("Updating database schema using migrations from %s", options.MigrationsURL)
		if err := ps.updateSchema(options.MigrationsURL); err != nil {
			log.D().Panicln("Could not update database schema:", err)
		}
	}
	return err
}

func (ps *postgresStorage) Close() error {
	ps.checkOpen()
	return ps.db.Close()
}

func (ps *postgresStorage) updateSchema(migrationsURL string) error {
	driver, err := migratepg.WithInstance(ps.db.DB, &migratepg.Config{})
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

func (ps *postgresStorage) checkOpen() {
	if ps.db == nil {
		log.D().Panicln("Repository is not yet Open")
	}
}
