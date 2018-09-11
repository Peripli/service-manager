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
	"github.com/jmoiron/sqlx"
)

type platformStorage struct {
	db *sqlx.DB
}

func (ps *platformStorage) Create(ctx context.Context, platform *types.Platform) error {
	return create(ctx, ps.db, platformTable, convertPlatformToDTO(platform))
}

func (ps *platformStorage) Get(ctx context.Context, id string) (*types.Platform, error) {
	platform := &Platform{}
	if err := get(ctx, ps.db, id, platformTable, platform); err != nil {
		return nil, err
	}
	return platform.Convert(), nil
}

func (ps *platformStorage) GetAll(ctx context.Context) ([]*types.Platform, error) {
	var platformDTOs []Platform
	err := getAll(ctx, ps.db, platformTable, &platformDTOs)
	if err != nil || len(platformDTOs) == 0 {
		return []*types.Platform{}, err
	}
	var platforms = make([]*types.Platform, 0, len(platformDTOs))
	for _, platformDTO := range platformDTOs {
		platforms = append(platforms, platformDTO.Convert())
	}
	return platforms, nil
}

func (ps *platformStorage) Delete(ctx context.Context, id string) error {
	return delete(ctx, ps.db, id, platformTable)
}

func (ps *platformStorage) Update(ctx context.Context, platform *types.Platform) error {
	return update(ctx, ps.db, platformTable, convertPlatformToDTO(platform))
}
