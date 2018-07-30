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
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/jmoiron/sqlx"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (bs *brokerStorage) Create(broker *types.Broker) error {
	return create(bs.db, brokerTable, convertBrokerToDTO(broker))

}

func (bs *brokerStorage) Get(id string) (*types.Broker, error) {
	broker := &Broker{}
	if err := get(bs.db, id, brokerTable, broker); err != nil {
		return nil, err
	}
	return broker.Convert(), nil
}

func (bs *brokerStorage) GetAll() ([]*types.Broker, error) {
	brokerDTOs := []Broker{}
	err := getAll(bs.db, brokerTable, &brokerDTOs)
	if err != nil || len(brokerDTOs) == 0 {
		return []*types.Broker{}, err
	}
	brokers := make([]*types.Broker, 0, len(brokerDTOs))
	for _, broker := range brokerDTOs {
		brokers = append(brokers, broker.Convert())
	}
	return brokers, nil
}

func (bs *brokerStorage) Delete(id string) error {
	return delete(bs.db, id, brokerTable)
}

func (bs *brokerStorage) Update(broker *types.Broker) error {
	return update(bs.db, brokerTable, convertBrokerToDTO(broker))
}
