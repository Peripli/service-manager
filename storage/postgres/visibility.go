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
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/selection"

	"github.com/Peripli/service-manager/pkg/types"
)

type visibilityStorage struct {
	db pgDB
}

func (vs *visibilityStorage) Create(ctx context.Context, visibility *types.Visibility) error {
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
	query := fmt.Sprintf(`SELECT 
		%[1]s.*,
		%[2]s.id "%[2]s.id",
		%[2]s.key "%[2]s.key",
		%[2]s.val "%[2]s.val",
		%[2]s.created_at "%[2]s.created_at",
		%[2]s.updated_at "%[2]s.updated_at",
		%[2]s.visibility_id "%[2]s.visibility_id"
	FROM %[1]s 
	LEFT JOIN %[2]s ON %[1]s.id = %[2]s.visibility_id`, visibilityTable, visibilityLabelsTable)

	query, queryParams, err := buildListQueryWithParams(query, visibilityTable, visibilityLabelsTable, criteria)
	if err != nil {
		return []*types.Visibility{}, err
	}
	query = vs.db.Rebind(query)

	log.C(ctx).Debugf("Executing query %s", query)
	rows, err := vs.db.QueryxContext(ctx, query, queryParams...)
	defer func() {
		if err := rows.Close(); err != nil {
			log.C(ctx).Errorf("Could not release connection when checking database s. Error: %s", err)
		}
	}()
	if err != nil {
		return nil, checkSQLNoRows(err)
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

func (vs *visibilityStorage) ListByPlatformID(ctx context.Context, platformID string) ([]*types.Visibility, error) {
	var visibilities []Visibility
	err := list(ctx, vs.db, visibilityTable, map[string]string{"platform_id": platformID}, &visibilities)
	if err != nil || len(visibilities) == 0 {
		return nil, err
	}
	visibilityDTOs := make([]*types.Visibility, 0, len(visibilities))
	for _, v := range visibilities {
		visibilityDTOs = append(visibilityDTOs, v.ToDTO())
	}
	return visibilityDTOs, nil
}

func (vs *visibilityStorage) Delete(ctx context.Context, id string) error {
	return remove(ctx, vs.db, id, visibilityTable)
}

func (vs *visibilityStorage) Update(ctx context.Context, visibility *types.Visibility) error {
	v := &Visibility{}
	v.FromDTO(visibility)
	return update(ctx, vs.db, visibilityTable, v)
}
