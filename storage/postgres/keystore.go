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
	"fmt"
	"time"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/jmoiron/sqlx"
)

const securityLockIndex = 111

// Safe represents a secret entity
type Safe struct {
	Secret    []byte    `db:"secret"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (s *Safe) GetID() string {
	return ""
}

func (s *Safe) ToObject() types.Object {
	return nil
}

func (s *Safe) FromObject(object types.Object) (storage.Entity, bool) {
	return nil, false
}

func (s *Safe) BuildLabels(labels types.Labels, newLabel func(id, key, value string) storage.Label) ([]storage.Label, error) {
	return nil, nil
}

func (s *Safe) NewLabel(id, key, value string) storage.Label {
	return nil
}

func (s *Safe) TableName() string {
	return "safe"
}

func (s *Safe) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	return nil, nil
}

func (s *Safe) LabelEntity() PostgresLabel {
	return nil
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (s *Storage) Lock(ctx context.Context) error {
	s.checkOpen()

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.isLocked {
		return fmt.Errorf("lock is already acquired")
	}
	if _, err := s.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", securityLockIndex); err != nil {
		return err
	}
	s.isLocked = true

	return nil
}

// Unlock releases the database lock.
func (s *Storage) Unlock(ctx context.Context) error {
	s.checkOpen()

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.isLocked {
		return nil
	}

	if _, err := s.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", securityLockIndex); err != nil {
		return err
	}
	s.isLocked = false

	return nil
}

// GetEncryptionKey returns the encryption key used to encrypt the credentials for brokers
func (s *Storage) GetEncryptionKey(ctx context.Context, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) ([]byte, error) {
	s.checkOpen()

	safe := &Safe{}
	rows, err := s.queryBuilder.NewQuery(safe).List(ctx)

	defer closeRows(ctx, rows)
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		if err := rows.StructScan(safe); err != nil {
			return nil, err
		}
	}
	if len(safe.Secret) == 0 {
		return []byte{}, nil
	}
	encryptedKey := []byte(safe.Secret)

	return transformationFunc(ctx, encryptedKey, s.layerOneEncryptionKey)
}

// SetEncryptionKey Sets the encryption key by encrypting it beforehand with the encryption key in the environment
func (s *Storage) SetEncryptionKey(ctx context.Context, key []byte, transformationFunc func(context.Context, []byte, []byte) ([]byte, error)) error {
	s.checkOpen()

	existingKey := &Safe{}
	rows, err := s.queryBuilder.NewQuery(existingKey).List(ctx)

	defer closeRows(ctx, rows)
	if err != nil {
		return err
	}
	if rows.Next() {
		if err := rows.StructScan(existingKey); err != nil {
			return err
		}
		if existingKey.Secret != nil && len(existingKey.Secret) > 0 {
			return fmt.Errorf("encryption key is already set")
		}
	}
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
