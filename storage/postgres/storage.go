/*
 *    Copyright 2018 The Service Manager Authors
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
	"sync"

	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/storage"
	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Storage returns a PostgreSQL storage
func Storage() storage.Storage {
	return &postgresStorage{}
}

type postgresStorage struct {
	once sync.Once
	db   *sqlx.DB
}

func (storage *postgresStorage) Open() error {
	var err error
	storage.once.Do(func() {
		uri, ok := env.Get("storage.uri").(string)
		if !ok {
			logrus.Panicf("Could not open connection for provided uri %s from postgres storage provider", uri)
		}
		storage.db, err = sqlx.Open("postgres", uri)
	})
	return err
}

func (storage *postgresStorage) Close() error {
	return storage.db.Close()
}
