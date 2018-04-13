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
	"github.com/Peripli/service-manager/storage"
	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
)

type postgresStorage struct {
	brokerStorage   storage.Broker
	platformStorage storage.Platform
}

func (p *postgresStorage) Broker() storage.Broker {
	return p.brokerStorage
}

func (p *postgresStorage) Platform() storage.Platform {
	return p.platformStorage
}

func newStorage(db *sqlx.DB) (storage.Storage, error) {
	return &postgresStorage{
		brokerStorage:   &brokerStorage{db},
		platformStorage: &platformStorage{db},
	}, nil
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
