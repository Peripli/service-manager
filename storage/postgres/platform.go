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
	"database/sql"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/rest"
	store "github.com/Peripli/service-manager/storage"
	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type platformStorage struct {
	db *sqlx.DB
}

const (
	// platformTable db table name for platforms
	platformTable = "platforms"

	// table db table name for credentials
	credentialsTable = "credentials"

	basicCredentialsType = 1
)

var (
	// selectByID selects platform by id
	selectByID = "SELECT id, type, name, description, created_at, updated_at FROM " + platformTable + " WHERE id=$1"

	// selectAll selects all platforms
	selectAll = "SELECT id, type, name, description, created_at, updated_at FROM " + platformTable

	// insertCredentials insert new credentials
	insertCredentials = "INSERT INTO " + credentialsTable + "(type, username, password) VALUES (:type, :username, :password) RETURNING id"

	// insertPlatform insert new platform
	insertPlatform = "INSERT INTO " + platformTable + "(id, type, name, description, credentials_id, created_at, updated_at) VALUES(:id, :type, :name, :description, :credentials_id, :created_at, :updated_at)"

	// deletePlatform deletes platform and corresponding credentials
	deletePlatform = fmt.Sprintf(`WITH pl AS (
		DELETE FROM %s
		WHERE
			id = $1
		RETURNING credentials_id
	)
	DELETE FROM %s
	WHERE id IN (SELECT credentials_id from pl)`, platformTable, credentialsTable)

	updatePlatform = "UPDATE " + platformTable + " SET name=:name, type=:type, description=:description, updated_at=:updated_at WHERE id=:id"
)

func (storage *platformStorage) Create(platform *rest.Platform) error {
	return transaction(storage.db, func(tx *sqlx.Tx) error {
		stmt, err := tx.PrepareNamed(insertCredentials)
		if err != nil {
			return err
		}
		var credentialsID int
		err = stmt.Get(&credentialsID, &Credentials{
			Type:     basicCredentialsType,
			Username: platform.Credentials.Basic.Username,
			Password: platform.Credentials.Basic.Password,
		})
		if err != nil {
			return err
		}

		_, err = tx.NamedExec(insertPlatform, &Platform{
			ID:            platform.ID,
			Name:          platform.Name,
			Type:          platform.Type,
			Description:   platform.Description,
			CredentialsID: int(credentialsID),
			CreatedAt:     platform.CreatedAt,
			UpdatedAt:     platform.UpdatedAt,
		})
		if err != nil {
			pqErr, ok := err.(*pq.Error)
			if !ok {
				return err
			}
			if pqErr.Code.Name() == "unique_violation" {
				logrus.Debug(pqErr)
				return store.ErrUniqueViolation
			}
		}
		return nil
	})
}

func (storage *platformStorage) Get(id string) (*rest.Platform, error) {
	platform := Platform{}
	err := storage.db.Get(&platform, selectByID, id)
	if err == sql.ErrNoRows {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rest.Platform{
		ID:          platform.ID,
		Name:        platform.Name,
		Type:        platform.Type,
		Description: platform.Description,
		CreatedAt:   platform.CreatedAt,
		UpdatedAt:   platform.UpdatedAt,
	}, nil
}

func restPlatformFromDTO(platform *Platform) *rest.Platform {
	return &rest.Platform{
		ID:          platform.ID,
		Type:        platform.Type,
		Name:        platform.Name,
		Description: platform.Description,
		CreatedAt:   platform.CreatedAt,
		UpdatedAt:   platform.UpdatedAt,
	}
}

func (storage *platformStorage) GetAll() ([]rest.Platform, error) {
	platformDTOs := []Platform{}
	err := storage.db.Select(&platformDTOs, selectAll)
	if err != nil || len(platformDTOs) == 0 {
		return []rest.Platform{}, err
	}
	var platforms = make([]rest.Platform, 0, len(platformDTOs))
	for _, platformDTO := range platformDTOs {
		platforms = append(platforms, *restPlatformFromDTO(&platformDTO))
	}
	return platforms, nil
}

func (storage *platformStorage) Delete(id string) error {
	return transaction(storage.db, func(tx *sqlx.Tx) error {
		result, err := tx.Exec(deletePlatform, &id)
		if err != nil {
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			return store.ErrNotFound
		}
		return nil
	})

}

func (storage *platformStorage) Update(platform *rest.Platform) error {
	updateQuery := platformUpdateQueryString(platform)
	if updateQuery == "" {
		logrus.Debug("Platform update: nothing to update")
		return nil
	}

	result, err := storage.db.NamedExec(updatePlatform, &Platform{
		ID:          platform.ID,
		Type:        platform.Type,
		Name:        platform.Name,
		Description: platform.Description,
		UpdatedAt:   platform.UpdatedAt,
	})
	if err != nil {
		sqlErr, ok := err.(*pq.Error)
		if ok && sqlErr.Code.Name() == "unique_violation" {
			return store.ErrUniqueViolation
		}
		return err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affectedRows != 1 {
		return store.ErrNotFound
	}
	return nil
}

func platformUpdateQueryString(platform *rest.Platform) string {
	set := make([]string, 0, 4)
	if platform.Name != "" {
		set = append(set, "name = :name")
	}
	if platform.Type != "" {
		set = append(set, "type = :type")
	}
	if platform.Description != "" {
		set = append(set, "description = :description")
	}
	if len(set) == 0 {
		return ""
	}
	set = append(set, "updated_at = :updated_at")
	return fmt.Sprintf("UPDATE %s SET %s WHERE id = :id", platformTable, strings.Join(set, ", "))
}
