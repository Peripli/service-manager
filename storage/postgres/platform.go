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
	"github.com/Peripli/service-manager/rest"
	"github.com/jmoiron/sqlx"
)

type platformStorage struct {
	db *sqlx.DB
}

const (
	// schema db schema name
	schema = `"SERVICE_MANAGER"`

	// table db table name for platforms
	table = "platform"

	basicCredentialsType = 1
)

var (
	// selectByID selects platform by id
	selectByID = "SELECT * FROM " + schema + "." + table + "WHERE id=$1"

	// selectByName selects platform by name
	selectByName = "SELECT * FROM " + schema + "." + table + "WHERE name=$1"

	// selectAll selects all platforms
	selectAll = "SELECT * FROM " + schema + "." + table

	// insertCredentials insert new credentials
	insertCredentials = "INSERT INTO " + schema + "." + table + "(type, username, password) VALUES (:type, :username, :password)"

	// insertPlatform insert new platform
	insertPlatform = "INSERT INTO " + schema + "." + table + "(id, type, name, description, credentials_id, created_at, updated_at) VALUES(:id, :type, :name, :description, :credentials_id, :created_at, :updated_at)"
)

func (storage *platformStorage) Create(platform *rest.Platform) error {
	// tx := storage.db.MustBegin()
	// credentials.Type = basicCredentialsType
	// rs := tx.MustExec(insertCredentials, credentials)
	// credentialsID, err := rs.LastInsertId()
	// if err != nil {
	// 	rollbackErr := tx.Rollback()
	// 	if rollbackErr != nil {
	// 		return fmt.Errorf("Insert error: %s; Rollback error: %s", err.Error(), rollbackErr.Error())
	// 	}
	// 	return err
	// }
	// platform.CredentialsID = int(credentialsID)
	// tx.MustExec(insertPlatform, platform)
	// return tx.Commit()

	return nil
}

func (storage *platformStorage) get(stmt string, arg string) (*rest.Platform, error) {
	// platform := dto.Platform{}
	// err := storage.db.Get(&platform, stmt, arg)
	// if err == sql.ErrNoRows {
	// 	return nil, nil
	// }
	// if err != nil {
	// 	return nil, err
	// }
	// return &platform, nil

	return nil, nil
}

func (storage *platformStorage) GetByID(id string) (*rest.Platform, error) {
	return storage.get(selectByID, id)
}

func (storage *platformStorage) GetByName(name string) (*rest.Platform, error) {
	return storage.get(selectByName, name)
}

func (storage *platformStorage) GetAll() ([]rest.Platform, error) {
	// platforms := []dto.Platform{}
	// err := storage.db.Select(&platforms, selectAll)
	// return platforms, err

	return []rest.Platform{}, nil
}

func (storage *platformStorage) Delete(id string) error {
	return nil
}

func (storage *platformStorage) Update(platform *rest.Platform) error {
	return nil
}
