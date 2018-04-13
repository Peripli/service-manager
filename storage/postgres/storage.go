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

 // package postgres implements the Service Manager storage interfaces for Postgresql DB
package postgres

import (
	"sync"

	"fmt"

	"github.com/Peripli/service-manager/storage"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

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

func (storage *postgresStorage) Open(uri string) error {
	var err error
	if uri == "" {
		return fmt.Errorf("Storage URI cannot be empty")
	}
	storage.once.Do(func() {
		storage.db, err = sqlx.Open(Storage, uri)
	})
	return err
}

func (storage *postgresStorage) Close() error {
	return storage.db.Close()
}
