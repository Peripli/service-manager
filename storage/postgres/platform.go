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
	// schema db schema name
	schema = `"SERVICE_MANAGER"`

	// table db table name for platforms
	table = "platforms"

	// table db table name for credentials
	credentialsTable = "credentials"

	basicCredentialsType = 1
)

var (
	// selectByID selects platform by id
	selectByID = "SELECT id, type, name, description, created_at, updated_at FROM " + schema + "." + table + " WHERE id=$1"

	// selectByName selects platform by name
	selectByName = "SELECT id, type, name, description, created_at, updated_at FROM " + schema + "." + table + " WHERE name=$1"

	// selectAll selects all platforms
	selectAll = "SELECT id, type, name, description, created_at, updated_at FROM " + schema + "." + table

	// insertCredentials insert new credentials
	insertCredentials = "INSERT INTO " + schema + "." + credentialsTable + "(type, username, password) VALUES (:type, :username, :password) RETURNING id"

	// insertPlatform insert new platform
	insertPlatform = "INSERT INTO " + schema + "." + table + "(id, type, name, description, credentials_id, created_at, updated_at) VALUES(:id, :type, :name, :description, :credentials_id, :created_at, :updated_at)"

	// deletePlatform deletes platform and corresponding credentials
	deletePlatform = `WITH pl AS (
		DELETE FROM "SERVICE_MANAGER".platforms
		WHERE
			id = $1
		RETURNING credentials_id
	)
	DELETE FROM "SERVICE_MANAGER".credentials
	WHERE id IN (SELECT credentials_id from pl)`

	updatePlatform = "UPDATE " + schema + "." + table + " SET name=:name, type=:type, description=:description, updated_at=:updated_at WHERE id=:id"
)

func handleRollback(transaction *sqlx.Tx, lastOperation error) error {
	rollbackErr := transaction.Rollback()
	if rollbackErr != nil {
		return fmt.Errorf("Rollback error: %s; Reason: %s", rollbackErr.Error(), lastOperation.Error())
	}
	return lastOperation
}

func (storage *platformStorage) Create(platform *rest.Platform) error {
	tx := storage.db.MustBegin()
	stmt, err := tx.PrepareNamed(insertCredentials)
	if err != nil {
		return handleRollback(tx, err)
	}
	var credentialsID int
	err = stmt.Get(&credentialsID, &Credentials{
		Type:     basicCredentialsType,
		Username: platform.Credentials.Basic.Username,
		Password: platform.Credentials.Basic.Password,
	})
	if err != nil {
		return handleRollback(tx, err)
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
			return handleRollback(tx, err)
		}
		if pqErr.Code.Name() == "unique_violation" {
			logrus.Debug(pqErr)
			return handleRollback(tx, store.ConflictEntityError)
		}
	}

	return tx.Commit()
}

func (storage *platformStorage) get(stmt string, arg string) (*rest.Platform, error) {
	platform := Platform{}
	err := storage.db.Get(&platform, stmt, arg)
	if err == sql.ErrNoRows {
		return nil, nil
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

func (storage *platformStorage) GetByID(id string) (*rest.Platform, error) {
	return storage.get(selectByID, id)
}

func (storage *platformStorage) GetByName(name string) (*rest.Platform, error) {
	return storage.get(selectByName, name)
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
	tx := storage.db.MustBegin()
	result, err := tx.Exec(deletePlatform, &id)
	if err != nil {
		return handleRollback(tx, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return handleRollback(tx, err)
	}

	if rowsAffected != 1 {
		return handleRollback(tx, store.MissingEntityError)
	}
	return tx.Commit()
}

func (storage *platformStorage) Update(platform *rest.Platform) error {
	result, err := storage.db.NamedExec(updatePlatform, &Platform{
		ID:          platform.ID,
		Type:        platform.Type,
		Name:        platform.Name,
		Description: platform.Description,
		UpdatedAt:   platform.UpdatedAt,
	})
	if err != nil {
		return err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affectedRows != 1 {
		return store.MissingEntityError
	}
	return nil
}
