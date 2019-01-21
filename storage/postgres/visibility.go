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
	"time"

	"github.com/Peripli/service-manager/pkg/util"

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

func (vs *visibilityStorage) createLabels(ctx context.Context, visibilityID string, labels types.Labels) error {
	vls := visibilityLabels{}
	if err := vls.FromDTO(visibilityID, labels); err != nil {
		return err
	}
	if err := vls.Validate(); err != nil {
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
	byID := query.ByField(query.EqualsOperator, "id", id)
	visibilities, err := vs.List(ctx, byID)
	if err != nil {
		return nil, err
	}
	if len(visibilities) == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	return visibilities[0], nil
}

func (vs *visibilityStorage) List(ctx context.Context, criteria ...query.Criterion) ([]*types.Visibility, error) {
	rows, err := listWithLabelsByCriteria(ctx, vs.db, Visibility{}, &VisibilityLabel{}, visibilityTable, criteria)
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

	visibilities := make(map[string]*types.Visibility)
	labels := make(map[string]map[string][]string)
	result := make([]*types.Visibility, 0)
	for rows.Next() {
		row := struct {
			*Visibility
			*VisibilityLabel `db:"visibility_labels"`
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		visibility, ok := visibilities[row.Visibility.ID]
		if !ok {
			visibility = row.Visibility.ToDTO()
			visibilities[row.Visibility.ID] = visibility
			result = append(result, visibility)
		}
		if labels[visibility.ID] == nil {
			labels[visibility.ID] = make(map[string][]string)
		}
		labels[visibility.ID][row.VisibilityLabel.Key.String] = append(labels[visibility.ID][row.VisibilityLabel.Key.String], row.VisibilityLabel.Val.String)
	}

	for _, vis := range result {
		vis.Labels = labels[vis.ID]
	}

	return result, nil
}

func (vs *visibilityStorage) Delete(ctx context.Context, criteria ...query.Criterion) error {
	return deleteAllByFieldCriteria(ctx, vs.db, visibilityTable, Visibility{}, criteria)
}

func (vs *visibilityStorage) Update(ctx context.Context, visibility *types.Visibility, labelChanges ...*query.LabelChange) error {
	v := &Visibility{}
	v.FromDTO(visibility)
	if err := update(ctx, vs.db, visibilityTable, v); err != nil {
		return err
	}
	if err := vs.updateLabels(ctx, v.ID, labelChanges); err != nil {
		return err
	}
	byVisibilityID := query.ByField(query.EqualsOperator, "visibility_id", v.ID)
	var labels []*VisibilityLabel
	if err := listByFieldCriteria(ctx, vs.db, visibilityLabelsTable, &labels, []query.Criterion{byVisibilityID}); err != nil {
		return err
	}
	visibilityLabels := visibilityLabels(labels)
	visibility.Labels = visibilityLabels.ToDTO()
	return nil
}

func (vs *visibilityStorage) updateLabels(ctx context.Context, visibilityID string, updateActions []*query.LabelChange) error {
	now := time.Now()
	newLabelFunc := func(labelID string, labelKey string, labelValue string) Labelable {
		return &VisibilityLabel{
			ID:                  toNullString(labelID),
			Key:                 toNullString(labelKey),
			Val:                 toNullString(labelValue),
			ServiceVisibilityID: toNullString(visibilityID),
			CreatedAt:           &now,
			UpdatedAt:           &now,
		}
	}
	return updateLabelsAbstract(ctx, newLabelFunc, vs.db, visibilityID, updateActions)
}
