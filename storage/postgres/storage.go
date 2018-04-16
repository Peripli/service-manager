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

// Package postgres implements the Service Manager storage interfaces for Postgresql DB
package postgres

import (
	"errors"
	"path"
	"runtime"
	"sync"

	"fmt"

	"github.com/Peripli/service-manager/storage"
	"github.com/golang-migrate/migrate"
	migratepg "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Storage defines the name of the PostgreSQL relational storage
const Storage = "postgres"

func init() {
	storage.Register(Storage, &postgresStorage{})
}

type postgresStorage struct {
	once sync.Once
	db   *sqlx.DB
}

func (storage *postgresStorage) Broker() storage.Broker {
	if storage.db == nil {
		panic("Storage is not yet Open")
	}
	return &brokerStorage{storage.db}
}

func (storage *postgresStorage) Platform() storage.Platform {
	if storage.db == nil {
		panic("Storage is not yet Open")
	}
	return &platformStorage{storage.db}
}

func (storage *postgresStorage) Open(uri string) error {
	var err error
	if uri == "" {
		return fmt.Errorf("storage URI cannot be empty")
	}
	storage.once.Do(func() {
		storage.db, err = sqlx.Connect(Storage, uri)
		if err != nil {
			logrus.Panicln("Could not connect to PostgreSQL:", err)
		}

		logrus.Debug("Updating database schema")
		if err := updateSchema(storage.db); err != nil {
			logrus.Panicln("Could not update database schema:", err)
		}

	})
	return err
}

func (storage *postgresStorage) Close() error {
	return storage.db.Close()
}

func getMigrateDir() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("Could not get database migrations scripts dir")
	}
	return path.Join(path.Dir(filename), "migrations"), nil
}

func updateSchema(db *sqlx.DB) error {
	driver, err := migratepg.WithInstance(db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}
	migrateDir, err := getMigrateDir()
	logrus.Debug("Migration scripts dir:", migrateDir)
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(migrateDir, "postgres", driver)
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

func transaction(db *sqlx.DB, f func(tx *sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		logrus.Error("Could not create transaction")
		return err
	}
	ok := false
	defer func() {
		if !ok {
			if txError := tx.Rollback(); txError != nil {
				logrus.Error("Could not rollback transaction", txError)
			}
		}
	}()

	if err = f(tx); err != nil {
		return err
	}
	ok = true
	return tx.Commit()
}
