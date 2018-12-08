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
	"context"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/selection"

	"github.com/Peripli/service-manager/pkg/types"
)

type visibilityStorage struct {
	db pgDB
}

func (vs *visibilityStorage) CreateLabels(ctx context.Context, visibilityID string, labels []*types.VisibilityLabel) error {
	for _, label := range labels {
		label.ServiceVisibilityID = visibilityID
		v := &VisibilityLabel{}
		v.FromDTO(label)
		if _, err := create(ctx, vs.db, visibilityLabelsTable, v); err != nil {
			return err
		}
	}
	return nil
}

func (vs *visibilityStorage) Create(ctx context.Context, visibility *types.Visibility) (string, error) {
	v := &Visibility{}
	v.FromDTO(visibility)
	return create(ctx, vs.db, visibilityTable, v)
}

func (vs *visibilityStorage) Get(ctx context.Context, id string) (*types.Visibility, error) {
	visibility := &Visibility{}
	if err := get(ctx, vs.db, id, visibilityTable, visibility); err != nil {
		return nil, err
	}
	return visibility.ToDTO(), nil
}

func (vs *visibilityStorage) List(ctx context.Context, criteria ...selection.Criterion) ([]*types.Visibility, error) {
	rows, err := listWithLabelsAndCriteria(ctx, vs.db, Visibility{}, VisibilityLabel{}, visibilityTable, visibilityLabelsTable, criteria)
	defer func() {
		if rows == nil {
			return
		}
		if err := rows.Close(); err != nil {
			log.C(ctx).Errorf("Could not release connection when checking database s. Error: %s", err)
		}
	}()
	if err != nil {
		return nil, checkIntegrityViolation(ctx, err)
		//return nil, checkSQLNoRows(err)
	}

	visibilities := make(map[string]*types.Visibility)
	result := make([]*types.Visibility, 0)
	for rows.Next() {
		row := struct {
			*Visibility
			*VisibilityLabel `db:"visibility_labels"`
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		if visibility, ok := visibilities[row.Visibility.ID]; !ok {
			visibility = row.Visibility.ToDTO()
			visibility.Labels = append(visibility.Labels, row.VisibilityLabel.ToDTO())

			visibilities[row.Visibility.ID] = visibility
			result = append(result, visibility)
		} else {
			visibility.Labels = append(visibility.Labels, row.VisibilityLabel.ToDTO())
		}
	}
	return result, nil
}

func (vs *visibilityStorage) Delete(ctx context.Context, id string) error {
	return remove(ctx, vs.db, id, visibilityTable)
}

func (vs *visibilityStorage) Update(ctx context.Context, visibility *types.Visibility) error {
	v := &Visibility{}
	v.FromDTO(visibility)
	return update(ctx, vs.db, visibilityTable, v)
}
