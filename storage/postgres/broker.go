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
	"github.com/Peripli/service-manager/storage/dto"
	"github.com/jmoiron/sqlx"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (storage *brokerStorage) Create(broker *dto.Broker) error {
	return nil
}

func (storage *brokerStorage) Get(id string) (*dto.Broker, error) {
	return nil, nil
}

func (storage *brokerStorage) GetAll() ([]*dto.Broker, error) {
	return []*dto.Broker{}, nil
}

func (storage *brokerStorage) Delete(id string) error {
	return nil
}

func (storage *brokerStorage) Update(broker *dto.Broker) error {
	return nil
}
