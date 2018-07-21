/*
 *    Copyright 2018 The Service Manager Authors
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
	"github.com/sirupsen/logrus"
)

type platformStorage struct {
	db *sqlx.DB
}

func (ps *platformStorage) Create(platform *types.Platform) error {
	return transaction(ps.db, func(tx *sqlx.Tx) error {
		stmt, err := tx.PrepareNamed(
			"INSERT INTO " + credentialsTable + "(type, username, password) VALUES (:type, :username, :password) RETURNING id")
		if err != nil {
			return err
		}
		var credentialsID int
		err = stmt.Get(&credentialsID, convertCredentialsToDTO(platform.Credentials))
		if err != nil {
			return err
		}

		platformDTO := convertPlatformToDTO(platform)
		platformDTO.CredentialsID = credentialsID
		_, err = tx.NamedExec(fmt.Sprintf(
			"INSERT INTO %s (id, type, name, description, credentials_id, created_at, updated_at) %s",
			platformTable,
			"VALUES(:id, :type, :name, :description, :credentials_id, :created_at, :updated_at)"),
			&platformDTO)
		return checkUniqueViolation(err)
	})
}

func (ps *platformStorage) Get(id string) (*types.Platform, error) {
	platform := Platform{}
	err := ps.db.Get(&platform,
		"SELECT id, type, name, description, created_at, updated_at FROM "+platformTable+" WHERE id=$1",
		id)
	if err = checkSQLNoRows(err); err != nil {
		return nil, err
	}
	return platform.Convert(), nil
}

func (ps *platformStorage) GetAll() ([]types.Platform, error) {
	platformDTOs := []Platform{}
	err := ps.db.Select(&platformDTOs,
		"SELECT id, type, name, description, created_at, updated_at FROM "+platformTable)
	if err != nil || len(platformDTOs) == 0 {
		return []types.Platform{}, err
	}
	var platforms = make([]types.Platform, 0, len(platformDTOs)+1)
	for _, platformDTO := range platformDTOs {
		platforms = append(platforms, *platformDTO.Convert())
	}
	return platforms, nil
}

func (ps *platformStorage) Delete(id string) error {
	// deletePlatform is a query that deletes platform and corresponding credentials
	deletePlatform := fmt.Sprintf(`WITH pl AS (
		DELETE FROM %s
		WHERE
			id = $1
		RETURNING credentials_id
	)
	DELETE FROM %s
	WHERE id IN (SELECT credentials_id from pl)`, platformTable, credentialsTable)

	return transaction(ps.db, func(tx *sqlx.Tx) error {
		result, err := tx.Exec(deletePlatform, &id)
		if err != nil {
			return err
		}
		return checkRowsAffected(result)
	})
}

func (ps *platformStorage) Update(platform *types.Platform) error {
	platformDTO := convertPlatformToDTO(platform)
	updateQuery, err := updateQuery(platformTable, platformDTO)
	if err != nil {
		return err
	}
	if updateQuery == "" {
		logrus.Debug("Platform update: nothing to update")
		return nil
	}
	result, err := ps.db.NamedExec(updateQuery, platformDTO)
	if err = checkUniqueViolation(err); err != nil {
		return err
	}
	return checkRowsAffected(result)
}
