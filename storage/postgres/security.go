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

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/security"
)

const securityLockIndex = 111

// Safe represents a secret entity
type Safe struct {
	Secret    []byte    `db:"secret"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (s *PostgresStorage) Lock(ctx context.Context) error {
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
func (s *PostgresStorage) Unlock(ctx context.Context) error {
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
func (s *PostgresStorage) GetEncryptionKey(ctx context.Context) ([]byte, error) {
	safe := &Safe{}
	rows, err := listByFieldCriteria(ctx, s.db, "safe", []query.Criterion{})
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
	return security.Decrypt(encryptedKey, s.encryptionKey)
}

// SetEncryptionKey Sets the encryption key by encrypting it beforehand with the encryption key in the environment
func (k *PostgresStorage) SetEncryptionKey(ctx context.Context, key []byte) error {
	rows, err := listByFieldCriteria(ctx, k.db, "safe", []query.Criterion{})
	defer closeRows(ctx, rows)
	if err != nil {
		return err
	}
	existingKey := &Safe{}
	if rows.Next() {
		if err := rows.StructScan(existingKey); err != nil {
			return err
		}
		if existingKey.Secret != nil && len(existingKey.Secret) > 0 {
			return fmt.Errorf("encryption key is already set")
		}
	}
	bytes, err := security.Encrypt(key, k.encryptionKey)
	if err != nil {
		return err
	}
	safe := Safe{
		Secret:    bytes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = create(ctx, k.db, "safe", safe)
	return err
}
