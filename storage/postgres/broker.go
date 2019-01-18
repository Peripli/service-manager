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
	"time"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
)

type brokerStorage struct {
	db pgDB
}

func (bs *brokerStorage) Create(ctx context.Context, broker *types.Broker) (string, error) {
	b := &Broker{}
	b.FromDTO(broker)
	id, err := create(ctx, bs.db, brokerTable, b)
	if err != nil {
		return "", err
	}
	return id, bs.createLabels(ctx, id, broker.Labels)
}

func (bs *brokerStorage) createLabels(ctx context.Context, brokerID string, labels types.Labels) error {
	vls := brokerLabels{}
	if err := vls.FromDTO(brokerID, labels); err != nil {
		return err
	}
	if err := vls.Validate(); err != nil {
		return err
	}
	for _, label := range vls {
		if _, err := create(ctx, bs.db, brokerLabelsTable, label); err != nil {
			return err
		}
	}
	return nil
}

func (bs *brokerStorage) Get(ctx context.Context, id string) (*types.Broker, error) {
	byID := query.ByField(query.EqualsOperator, "id", id)

	brokers, err := bs.List(ctx, byID)
	if err != nil {
		return nil, err
	}
	if len(brokers) == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	return brokers[0], nil
}

func (bs *brokerStorage) List(ctx context.Context, criteria ...query.Criterion) ([]*types.Broker, error) {
	rows, err := listWithLabelsByCriteria(ctx, bs.db, Broker{}, &BrokerLabel{}, brokerTable, criteria)
	defer func() {
		if rows == nil {
			return
		}
		if err := rows.Close(); err != nil {
			log.C(ctx).Errorf("Could not release connection when checking database. Error: %s", err)
		}
	}()
	if err != nil {
		return nil, err
	}

	brokers := make(map[string]*types.Broker)
	labels := make(map[string]map[string][]string)
	result := make([]*types.Broker, 0)
	for rows.Next() {
		row := struct {
			*Broker
			*BrokerLabel `db:"broker_labels"`
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		broker, ok := brokers[row.Broker.ID]
		if !ok {
			broker = row.Broker.ToDTO()
			brokers[row.Broker.ID] = broker
			result = append(result, broker)
		}
		if labels[broker.ID] == nil {
			labels[broker.ID] = make(map[string][]string)
		}
		labels[broker.ID][row.BrokerLabel.Key.String] = append(labels[broker.ID][row.BrokerLabel.Key.String], row.BrokerLabel.Val.String)
	}

	for _, b := range result {
		b.Labels = labels[b.ID]
	}

	return result, nil
}

func (bs *brokerStorage) Delete(ctx context.Context, criteria ...query.Criterion) error {
	return deleteAllByFieldCriteria(ctx, bs.db, brokerTable, Broker{}, criteria)

}

func (bs *brokerStorage) Update(ctx context.Context, broker *types.Broker, labelChanges ...*query.LabelChange) error {
	b := &Broker{}
	b.FromDTO(broker)
	if err := update(ctx, bs.db, brokerTable, b); err != nil {
		return err
	}
	if err := bs.updateLabels(ctx, b.ID, labelChanges); err != nil {
		return err
	}
	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", b.ID)
	var labels []*BrokerLabel
	if err := listByFieldCriteria(ctx, bs.db, brokerLabelsTable, &labels, []query.Criterion{byBrokerID}); err != nil {
		return err
	}
	brokerLabels := brokerLabels(labels)
	broker.Labels = brokerLabels.ToDTO()
	return nil
}

func (bs *brokerStorage) updateLabels(ctx context.Context, brokerID string, updateActions []*query.LabelChange) error {
	now := time.Now()
	newLabelFunc := func(labelID string, labelKey string, labelValue string) Labelable {
		return &BrokerLabel{
			ID:        toNullString(labelID),
			Key:       toNullString(labelKey),
			Val:       toNullString(labelValue),
			BrokerID:  toNullString(brokerID),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
	}
	return updateLabelsAbstract(ctx, newLabelFunc, bs.db, brokerID, updateActions)
}
