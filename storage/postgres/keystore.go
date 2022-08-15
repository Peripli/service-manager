/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.com/jmoiron/sqlx"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

const (
	securityLockIndex = 111
	SafeTable         = "safe"
)

// EncryptingLocker builds an encrypting storage.Locker with the pre-defined lock index
func EncryptingLocker(storage *Storage) storage.Locker {
	return &Locker{Storage: storage, AdvisoryIndex: securityLockIndex}
}

// Safe represents a secret entity
type Safe struct {
	Secret    []byte    `db:"secret"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (s *Safe) GetID() string {
	return ""
}

func (s *Safe) ToObject() (types.Object, error) {
	return nil, nil
}

func (s *Safe) FromObject(object types.Object) (storage.Entity, error) {
	return nil, nil
}

func (s *Safe) NewLabel(id, entityID, key, value string) storage.Label {
	return nil
}

func (s *Safe) TableName() string {
	return SafeTable
}

func (s *Safe) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	return nil, nil
}

func (s *Safe) LabelEntity() PostgresLabel {
	return nil
}

// GetEncryptionKey returns the encryption key used to encrypt the credentials for brokers
func (s *Storage) GetEncryptionKey(ctx context.Context, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) ([]byte, error) {
	s.checkOpen()

	safe := &Safe{}
	if err := s.db.GetContext(ctx, safe, "SELECT * FROM safe"); err != nil {
		if err == sql.ErrNoRows {
			return []byte{}, nil
		}
		return nil, err
	}
	encryptedKey := []byte(safe.Secret)

	return transformationFunc(ctx, encryptedKey, s.layerOneEncryptionKey)
}

// SetEncryptionKey Sets the encryption key by encrypting it beforehand with the encryption key in the environment
func (s *Storage) SetEncryptionKey(ctx context.Context, key []byte, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) error {
	s.checkOpen()

	bytes, err := transformationFunc(ctx, key, s.layerOneEncryptionKey)
	if err != nil {
		return err
	}

	err = create(ctx, s.db, "safe", &Safe{}, Safe{
		Secret:    bytes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	return err
}
