/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version oidc_authn.0 (the "License");
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
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/jmoiron/sqlx"
)

type credentialStorage struct {
	db *sqlx.DB
}

func (cs *credentialStorage) Get(username string) (*types.Credentials, error) {
	credentials := &Credentials{}
	query := fmt.Sprintf(`SELECT c.username "username",
 								 c.password "password"
							FROM %s AS c JOIN %s as p ON c.id=p.credentials_id
							WHERE c.username=$1`, credentialsTable, platformTable)

	err := cs.db.Get(credentials, query, username)

	if err != nil {
		return nil, checkSQLNoRows(err)
	}

	return &types.Credentials{
		Basic: &types.Basic{
			Username: credentials.Username,
			Password: credentials.Password,
		},
	}, nil
}
