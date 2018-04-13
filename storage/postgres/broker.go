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
	"github.com/Peripli/service-manager/types"
	"github.com/jmoiron/sqlx"
	"context"
)

type brokerStorage struct {
	db *sqlx.DB
}

func (storage *brokerStorage) Create(ctx context.Context, broker *types.Broker) error {
	return nil
}

func (storage *brokerStorage) Find(ctx context.Context, id string) (*types.Broker, error) {
	return &types.Broker{
		Name:     "brokerName",
		ID:       "brokerID",
		URL:      "http://localhost:8080/broker",
		User:     "brokerAdmin",
		Password: "brokerAdmin",
	}, nil
}

func (storage *brokerStorage) FindAll(ctx context.Context) ([]*types.Broker, error) {
	return []*types.Broker{}, nil
}

func (storage *brokerStorage) Delete(ctx context.Context, id string) error {
	return nil
}

func (storage *brokerStorage) Update(ctx context.Context, broker *types.Broker) error {
	return nil
}
