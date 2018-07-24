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

	"github.com/Peripli/service-manager/types"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (bs *brokerStorage) Create(broker *types.Broker) error {
	brokerDTO := convertBrokerToDTO(broker)
	query := fmt.Sprintf(
		"INSERT INTO %s (id, name, description, broker_url, created_at, updated_at, catalog, username, password) %s",
		brokerTable,
		"VALUES (:id, :name, :description, :broker_url, :created_at, :updated_at, :catalog, :username, :password)")
	_, err := bs.db.NamedExec(query, &brokerDTO)
	return checkUniqueViolation(err)
}

func (bs *brokerStorage) Get(id string) (*types.Broker, error) {
	broker := &Broker{}
	query := "SELECT * FROM " + brokerTable + " WHERE id=$1"
	err := bs.db.Get(broker, query, id)
	if err != nil {
		return nil, checkSQLNoRows(err)
	}
	return broker.Convert(), nil
}

func (bs *brokerStorage) GetAll() ([]*types.Broker, error) {
	brokerDTOs := []Broker{}
	query := "SELECT * FROM " + brokerTable
	err := bs.db.Select(&brokerDTOs, query)
	if err != nil || len(brokerDTOs) == 0 {
		return []*types.Broker{}, err
	}
	brokers := make([]*types.Broker, 0, len(brokerDTOs)+1)
	for _, broker := range brokerDTOs {
		brokers = append(brokers, broker.Convert())
	}
	return brokers, nil
}

func (bs *brokerStorage) Delete(id string) error {
	deleteBroker := fmt.Sprintf(`DELETE FROM %s WHERE id=$1`, brokerTable)

	result, err := bs.db.Exec(deleteBroker, &id)
	if err != nil {
		return err
	}
	return checkRowsAffected(result)
}

func (bs *brokerStorage) Update(broker *types.Broker) error {
	brokerDTO := convertBrokerToDTO(broker)
	updateQueryString, err := updateQuery(brokerTable, brokerDTO)
	if err != nil {
		return err
	}
	if updateQueryString == "" {
		logrus.Debug("Broker update: nothing to update")
		return nil
	}
	result, err := bs.db.NamedExec(updateQueryString, brokerDTO)
	if err = checkUniqueViolation(err); err != nil {
		return err
	}
	return checkRowsAffected(result)
}
