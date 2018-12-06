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
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"
)

const securityLockIndex = 111

type securityStorage struct {
	db            pgDB
	encryptionKey []byte
	isLocked      bool
	mutex         *sync.Mutex
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (s *securityStorage) Lock(ctx context.Context) error {
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
func (s *securityStorage) Unlock(ctx context.Context) error {
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

// Fetcher returns a KeyFetcher configured to fetch a key from the database
func (s *securityStorage) Fetcher() security.KeyFetcher {
	return &keyFetcher{s.db, []byte(s.encryptionKey)}
}

// Setter returns a KeySetter configured to set a key in the database
func (s *securityStorage) Setter() security.KeySetter {
	return &keySetter{s.db, s.encryptionKey}
}

type keyFetcher struct {
	db            pgDB
	encryptionKey []byte
}

// GetEncryptionKey returns the encryption key used to encrypt the credentials for brokers
func (s *keyFetcher) GetEncryptionKey(ctx context.Context) ([]byte, error) {
	var safes []Safe
	if err := list(ctx, s.db, "safe", map[string][]string{}, &safes); err != nil {
		return nil, err
	}
	if len(safes) != 1 {
		log.C(ctx).Warnf("Unexpected number of keys found: %d", len(safes))
		return []byte{}, nil
	}
	encryptedKey := []byte(safes[0].Secret)
	return security.Decrypt(encryptedKey, s.encryptionKey)
}

type keySetter struct {
	db            pgDB
	encryptionKey []byte
}

// Sets the encryption key by encrypting it beforehand with the encryption key in the environment
func (k *keySetter) SetEncryptionKey(ctx context.Context, key []byte) error {
	var safes []Safe
	if err := list(ctx, k.db, "safe", map[string][]string{}, &safes); err != nil {
		return err
	}
	if len(safes) != 0 {
		return fmt.Errorf("encryption key is already set")
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
