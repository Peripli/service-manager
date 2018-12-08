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
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

type credentialStorage struct {
	db pgDB
}

func (cs *credentialStorage) Get(ctx context.Context, username string) (*types.Credentials, error) {
	platform := &Platform{}
	query := fmt.Sprintf("SELECT * FROM %s WHERE username=$1", platformTable)
	log.C(ctx).Debugf("Executing query %s", query)

	err := cs.db.GetContext(ctx, platform, query, username)

	if err != nil {
		return nil, checkSQLNoRows(err)
	}

	bytes, err := json.Marshal(platform.ToDTO())
	if err != nil {
		return nil, err
	}
	return &types.Credentials{
		Basic: &types.Basic{
			Username: platform.Username,
			Password: platform.Password,
		},
		Details: bytes,
	}, nil
}
