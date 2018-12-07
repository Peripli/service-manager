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

	"github.com/Peripli/service-manager/pkg/types"
)

type visibilityStorage struct {
	db pgDB
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

func (vs *visibilityStorage) List(ctx context.Context) ([]*types.Visibility, error) {
	var visibilities []Visibility
	err := list(ctx, vs.db, visibilityTable, map[string][]string{}, &visibilities)
	if err != nil || len(visibilities) == 0 {
		return []*types.Visibility{}, err
	}
	var visibilityDTOs = make([]*types.Visibility, 0, len(visibilities))
	for _, visibility := range visibilities {
		visibilityDTOs = append(visibilityDTOs, visibility.ToDTO())
	}
	return visibilityDTOs, nil
}

func (vs *visibilityStorage) ListByPlatformID(ctx context.Context, platformID string) ([]*types.Visibility, error) {
	var visibilities []Visibility
	err := list(ctx, vs.db, visibilityTable, map[string][]string{"platform_id": {platformID, ""}}, &visibilities)
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
