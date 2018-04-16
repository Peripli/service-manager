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

package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type brokerStorage struct {
	db *sqlx.DB
}

type fetcher func(dest interface{}, query string, args ...interface{}) error

func (store *brokerStorage) Create(broker *rest.Broker) error {
	return transaction(store.db, func(tx *sqlx.Tx) error {
		credentialsDTO := convertCredentialsToDTO(broker.Credentials)
		statement, err := tx.PrepareNamed("INSERT INTO credentials (type, username, password, token) VALUES (:type, :username, :password, :token) RETURNING id")
		if err != nil {
			logrus.Error("Unable to create prepared statement")
			return err
		}

		var credentialsID int
		err = statement.Get(&credentialsID, credentialsDTO)
		if err != nil {
			logrus.Error("Prepared statement execution failed")
			return err
		}
		brokerDTO := convertBrokerToDTO(broker)
		brokerDTO.CredentialsID = credentialsID

		_, err = tx.NamedExec("INSERT INTO brokers (id, name, description, broker_url, created_at, updated_at, credentials_id) VALUES (:id, :name, :description, :broker_url, :created_at, :updated_at, :credentials_id)", brokerDTO)
		if err != nil {
			logrus.Error("Unable to insert broker")
			sqlErr, ok := err.(*pq.Error)
			if ok && sqlErr.Code.Name() == "unique_violation" {
				return storage.ErrUniqueViolation
			}
			return err
		}
		return nil
	})
}

func (store *brokerStorage) Get(id string) (*rest.Broker, error) {
	broker, err := retrieveBroker(store.db.Get, id)
	if err != nil {
		return nil, err
	}
	return broker.ToRestModel(), nil
}

func (store *brokerStorage) GetAll() ([]rest.Broker, error) {
	brokers := []Broker{}
	if err := store.db.Select(&brokers, "SELECT * FROM brokers"); err != nil {
		logrus.Error("An error occurred while retrieving all brokers")
		return nil, err
	}
	restBrokers := make([]rest.Broker, len(brokers))
	for i, val := range brokers {
		restBrokers[i] = *val.ToRestModel()
	}
	return restBrokers, nil
}

func (store *brokerStorage) Delete(id string) error {
	return transaction(store.db, func(tx *sqlx.Tx) error {
		broker, err := retrieveBroker(tx.Get, id)
		if err != nil {
			return err
		}

		_, err = tx.Exec("DELETE FROM brokers WHERE id = $1", id)
		if err != nil {
			logrus.Error("An error occurred while deleting broker with id:", id)
			return err
		}

		credentialsID := broker.CredentialsID
		_, err = tx.Exec("DELETE FROM credentials WHERE id = $1", credentialsID)
		if err != nil {
			logrus.Error("Could not delete broker credentials with id:", credentialsID)
			return err
		}
		return nil
	})
}

func (store *brokerStorage) Update(broker *rest.Broker) error {
	return transaction(store.db, func(tx *sqlx.Tx) error {
		updateQueryString := generateUpdateQueryString(broker)

		brokerDTO := convertBrokerToDTO(broker)
		if updateQueryString != "" {
			result, err := tx.NamedExec(updateQueryString, brokerDTO)
			if err != nil {
				logrus.Error("Unable to update broker")
				sqlErr, ok := err.(*pq.Error)
				if ok && sqlErr.Code.Name() == "unique_violation" {
					return storage.ErrUniqueViolation
				}
				return err
			}
			affectedRows, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affectedRows != 1 {
				return storage.ErrNotFound
			}
		}

		if broker.Credentials != nil {
			credentialsDTO := convertCredentialsToDTO(broker.Credentials)
			err := tx.Get(&credentialsDTO.ID, "SELECT credentials_id FROM brokers WHERE id = $1", broker.ID)
			if err != nil {
				logrus.Error("Unable to retrieve broker credentials")
				return err
			}
			_, err = tx.NamedExec(
				"UPDATE credentials SET type = :type, username = :username, password = :password, token = :token WHERE id = :id",
				credentialsDTO)
			if err != nil {
				logrus.Error("Unable to update credentials")
				return err
			}
		}
		return nil
	})
}

func generateUpdateQueryString(broker *rest.Broker) string {
	set := make([]string, 0, 5)
	if broker.Name != "" {
		set = append(set, "name = :name")
	}
	if broker.Description != "" {
		set = append(set, "description = :description")
	}
	if broker.BrokerURL != "" {
		set = append(set, "broker_url = :broker_url")
	}
	if len(set) == 0 {
		return ""
	}
	set = append(set, "updated_at = :updated_at")
	update := fmt.Sprintf("UPDATE brokers SET %s WHERE id = :id",
		strings.Join(set, ", "))

	return update
}

func retrieveBroker(fetch fetcher, id string) (*Broker, error) {
	broker := Broker{}
	if err := fetch(&broker, "SELECT * FROM brokers WHERE id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		logrus.Error("An error occurred while retrieving broker with id:", id)
		return nil, err
	}
	return &broker, nil
}
