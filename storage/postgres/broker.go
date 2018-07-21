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
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/jmoiron/sqlx"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (bs *brokerStorage) Create(broker *types.Broker) error {
	return transaction(bs.db, func(tx *sqlx.Tx) error {
		credentialsDTO := convertCredentialsToDTO(broker.Credentials)
		statement, err := tx.PrepareNamed(
			"INSERT INTO " + credentialsTable + " (type, username, password, token) VALUES (:type, :username, :password, :token) RETURNING id")
		if err != nil {
			return fmt.Errorf("unable to create credentials for broker: %s", err)
		}

		var credentialsID int
		err = statement.Get(&credentialsID, credentialsDTO)
		if err != nil {
			return fmt.Errorf("unable to create broker: %s", err)
		}
		brokerDTO := convertBrokerToDTO(broker)
		brokerDTO.CredentialsID = credentialsID

		_, err = tx.NamedExec(fmt.Sprintf(
			"INSERT INTO %s (id, name, description, broker_url, created_at, updated_at, credentials_id, catalog) %s",
			brokerTable,
			"VALUES (:id, :name, :description, :broker_url, :created_at, :updated_at, :credentials_id, :catalog)"),
			&brokerDTO)
		return checkUniqueViolation(err)
	})
}

func (bs *brokerStorage) Get(id string) (*types.Broker, error) {
	broker := &Broker{}
	query := fmt.Sprintf(`SELECT b.*, 
								c.username "c.username", 
								c.password "c.password",
								c.id "c.id"
						 FROM %s AS b INNER JOIN %s AS c ON b.credentials_id=c.id
						 WHERE b.id=$1`, brokerTable, credentialsTable)

	err := bs.db.Get(broker, query, id)

	if err != nil {
		return nil, checkSQLNoRows(err)
	}
	result := broker.Convert()
	return result, nil
}

func (bs *brokerStorage) GetAll() ([]types.Broker, error) {
	brokerDTOs := []Broker{}
	err := bs.db.Select(&brokerDTOs, "SELECT * FROM "+brokerTable)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve all brokers: %s", err)
	}
	brokers := make([]types.Broker, 0, len(brokerDTOs)+1)
	for _, broker := range brokerDTOs {
		brokers = append(brokers, *broker.Convert())
	}
	return brokers, nil
}

func (bs *brokerStorage) Delete(id string) error {
	// deleteBroker is a query that deletes Broker and corresponding credentials
	deleteBroker := fmt.Sprintf(`WITH br AS (
		DELETE FROM %s
		WHERE
			id = $1
		RETURNING credentials_id
	)
	DELETE FROM %s
	WHERE id IN (SELECT credentials_id from br)`, brokerTable, credentialsTable)

	return transaction(bs.db, func(tx *sqlx.Tx) error {
		result, err := tx.Exec(deleteBroker, &id)
		if err != nil {
			return fmt.Errorf("unable to delete broker: %s", err)
		}
		return checkRowsAffected(result)
	})
}

func (bs *brokerStorage) Update(broker *types.Broker) error {
	return transaction(bs.db, func(tx *sqlx.Tx) error {
		brokerDTO := convertBrokerToDTO(broker)

		updateQueryString, err := updateQuery(brokerTable, brokerDTO)
		if err != nil {
			return err
		}
		if updateQueryString != "" {
			result, err := tx.NamedExec(updateQueryString, brokerDTO)
			if err = checkUniqueViolation(err); err != nil {
				return err
			}
			if err = checkRowsAffected(result); err != nil {
				return err
			}
		}

		if broker.Credentials != nil {
			credentialsDTO := convertCredentialsToDTO(broker.Credentials)
			err := tx.Get(&credentialsDTO.ID, "SELECT credentials_id FROM "+brokerTable+" WHERE id = $1", broker.ID)
			if err != nil {
				return fmt.Errorf("unable to retrieve broker credentials: %s", err)
			}
			_, err = tx.NamedExec(
				"UPDATE "+credentialsTable+" SET type = :type, username = :username, password = :password, token = :token WHERE id = :id",
				credentialsDTO)
			if err != nil {
				return fmt.Errorf("unable to update credentials: %s", err)
			}
		}
		return nil
	})
}
