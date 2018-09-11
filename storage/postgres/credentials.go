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
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/jmoiron/sqlx"
)

type credentialStorage struct {
	db *sqlx.DB
}

func (cs *credentialStorage) Get(ctx context.Context, username string) (*types.Credentials, error) {
	platformCredentials := &Platform{}
	query := fmt.Sprintf("SELECT username, password FROM %s WHERE username=$1", platformTable)
	log.C(ctx).Debugf("Executing query %s", query)

	err := cs.db.GetContext(ctx, platformCredentials, query, username)

	if err != nil {
		return nil, checkSQLNoRows(err)
	}

	return types.NewBasicCredentials(platformCredentials.Username, platformCredentials.Password), nil
}
