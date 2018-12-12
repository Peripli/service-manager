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
	"database/sql"
	"time"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
)

type visibilityStorage struct {
	db pgDB
}

func (vs *visibilityStorage) Create(ctx context.Context, visibility *types.Visibility) (string, error) {
	v := &Visibility{}
	v.FromDTO(visibility)
	id, err := create(ctx, vs.db, visibilityTable, v)
	if err != nil {
		return "", err
	}
	return id, vs.createLabels(ctx, id, visibility.Labels)
}

func (vs *visibilityStorage) createLabels(ctx context.Context, visibilityID string, labels []*types.VisibilityLabel) error {
	for _, label := range labels {
		label.ServiceVisibilityID = visibilityID
	}
	vls := visibilityLabels{}
	if err := vls.FromDTO(labels); err != nil {
		return err
	}
	for _, label := range vls {
		if _, err := create(ctx, vs.db, visibilityLabelsTable, label); err != nil {
			return err
		}
	}
	return nil
}

func (vs *visibilityStorage) Get(ctx context.Context, id string) (*types.Visibility, error) {
	visibility := &Visibility{}
	if err := get(ctx, vs.db, id, visibilityTable, visibility); err != nil {
		return nil, err
	}
	return visibility.ToDTO(), nil
}

func (vs *visibilityStorage) List(ctx context.Context, criteria ...query.Criterion) ([]*types.Visibility, error) {
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
			hasMatchingLabelKey := false
			labelDTO := row.VisibilityLabel.ToDTO()
			for _, label := range visibility.Labels {
				if label.Key == labelDTO.Key {
					hasMatchingLabelKey = true
					label.Value = append(label.Value, labelDTO.Value...)
					break
				}
			}
			if !hasMatchingLabelKey {
				visibility.Labels = append(visibility.Labels, labelDTO)
			}
		}
	}

	return result, nil
}

func (vs *visibilityStorage) Delete(ctx context.Context, id string) error {
	return remove(ctx, vs.db, id, visibilityTable)
}

func (vs *visibilityStorage) Update(ctx context.Context, visibility *types.Visibility, labelChanges ...query.LabelChange) error {
	v := &Visibility{}
	v.FromDTO(visibility)
	if err := update(ctx, vs.db, visibilityTable, v); err != nil {
		return err
	}
	return vs.updateLabels(ctx, v.ID, labelChanges)
}

func (vs *visibilityStorage) updateLabels(ctx context.Context, visibilityID string, updateActions []query.LabelChange) error {
	now := time.Now()
	newLabelFunc := func(labelID string, labelKey string, labelValue string) Labelable {
		return &VisibilityLabel{
			ID:                  sql.NullString{String: labelID, Valid: labelID != ""},
			Key:                 sql.NullString{String: labelKey, Valid: labelKey != ""},
			Val:                 sql.NullString{String: labelValue, Valid: labelValue != ""},
			ServiceVisibilityID: sql.NullString{String: visibilityID, Valid: visibilityID != ""},
			CreatedAt:           &now,
			UpdatedAt:           &now,
		}
	}
	return updateLabels(ctx, newLabelFunc, vs.db, visibilityID, updateActions)
}
