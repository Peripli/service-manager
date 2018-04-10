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
	"database/sql"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (store *brokerStorage) Create(broker *rest.Broker) error {
	return nil
}

func (store *brokerStorage) Get(id string) (*rest.Broker, error) {
	broker := Broker{}
	if err := store.db.Get(&broker, "SELECT * FROM brokers WHERE id = $1", id); err != nil {
		logrus.Debug("An error occurred while retrieving broker with id:", id)
		if err == sql.ErrNoRows {
			return nil, storage.NotFoundError
		}
		return nil, err
	}

	return broker.ConvertToRestModel(), nil
}

func (store *brokerStorage) GetAll() ([]rest.Broker, error) {
	brokers := []rest.Broker{}
	if err := store.db.Get(&brokers, "SELECT * FORM brokers"); err != nil {
		logrus.Debug("An error occurred while retrieving all brokers")
		return nil, err
	}
	return brokers, nil
}

func (store *brokerStorage) Delete(id string) error {
	if _, err := store.db.Exec("DELETE FROM brokers WHERE id = $1", id); err != nil {
		logrus.Debug("An error occurred while deleting broker with id:", id)
		return err
	}
	return nil
}

func (store *brokerStorage) Update(broker *rest.Broker) error {
	//_, err := store.db.Exec("UPDATE brokers SET (id, name, description, created_at, updated_at, broker_url, credentials_id) = $1, $2, $3, $4, $5, $6, $7", broker.ID, broker.Name, broker.Description, broker.CreatedAt, broker.UpdatedAt, broker.BrokerURL, broker.CredentialsID)
	/*
		if err != nil {
			logrus.Debug("An error occurred while updating broker with id:", broker.ID)
			return err
		}
	*/
	return nil
}
