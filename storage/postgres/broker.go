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
	"context"

	"github.com/Peripli/service-manager/pkg/types"
)

type brokerStorage struct {
	db pgDB
}

func (bs *brokerStorage) Create(ctx context.Context, broker *types.Broker) (string, error) {
	b := &Broker{}
	b.FromDTO(broker)
	return create(ctx, bs.db, brokerTable, b)
}

func (bs *brokerStorage) Get(ctx context.Context, id string) (*types.Broker, error) {
	broker := &Broker{}
	if err := get(ctx, bs.db, id, brokerTable, broker); err != nil {
		return nil, err
	}
	return broker.ToDTO(), nil
}

func (bs *brokerStorage) List(ctx context.Context) ([]*types.Broker, error) {
	var brokerDTOs []Broker
	err := list(ctx, bs.db, brokerTable, map[string][]string{}, &brokerDTOs)
	if err != nil || len(brokerDTOs) == 0 {
		return []*types.Broker{}, err
	}
	brokers := make([]*types.Broker, 0, len(brokerDTOs))
	for _, broker := range brokerDTOs {
		brokers = append(brokers, broker.ToDTO())
	}
	return brokers, nil
}

func (bs *brokerStorage) Delete(ctx context.Context, id string) error {
	return remove(ctx, bs.db, id, brokerTable)
}

func (bs *brokerStorage) Update(ctx context.Context, broker *types.Broker) error {
	b := &Broker{}
	b.FromDTO(broker)
	return update(ctx, bs.db, brokerTable, b)
}
