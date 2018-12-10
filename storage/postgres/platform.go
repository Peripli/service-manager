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

type platformStorage struct {
	db pgDB
}

func (ps *platformStorage) Create(ctx context.Context, platform *types.Platform) (string, error) {
	p := &Platform{}
	p.FromDTO(platform)
	return create(ctx, ps.db, platformTable, p)
}

func (ps *platformStorage) Get(ctx context.Context, id string) (*types.Platform, error) {
	platform := &Platform{}
	if err := get(ctx, ps.db, id, platformTable, platform); err != nil {
		return nil, err
	}
	return platform.ToDTO(), nil
}

func (ps *platformStorage) List(ctx context.Context) ([]*types.Platform, error) {
	var platforms []Platform
	err := list(ctx, ps.db, platformTable, map[string][]string{}, &platforms)
	if err != nil || len(platforms) == 0 {
		return []*types.Platform{}, err
	}
	var platformDTOs = make([]*types.Platform, 0, len(platforms))
	for _, platform := range platforms {
		platformDTOs = append(platformDTOs, platform.ToDTO())
	}
	return platformDTOs, nil
}

func (ps *platformStorage) Delete(ctx context.Context, id string) error {
	return remove(ctx, ps.db, id, platformTable)
}

func (ps *platformStorage) Update(ctx context.Context, platform *types.Platform) error {
	p := &Platform{}
	p.FromDTO(platform)
	return update(ctx, ps.db, platformTable, p)
}
