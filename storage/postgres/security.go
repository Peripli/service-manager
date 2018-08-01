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
	"fmt"
	"time"

	"github.com/Peripli/service-manager/security"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type securityStorage struct {
	db            *sqlx.DB
	encryptionKey []byte
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
	db            *sqlx.DB
	encryptionKey []byte
}

// GetEncryptionKey returns the encryption key used to encrypt the credentials for brokers
func (s *keyFetcher) GetEncryptionKey() ([]byte, error) {
	var safes []Safe
	if err := getAll(s.db, "safe", &safes); err != nil {
		return nil, err
	}
	if len(safes) != 1 {
		logrus.Warnf("Unexpected number of keys found: %d", len(safes))
		return []byte{}, nil
	}
	encryptedKey := []byte(safes[0].Secret)
	return security.Decrypt(encryptedKey, s.encryptionKey)
}

type keySetter struct {
	db            *sqlx.DB
	encryptionKey []byte
}

// Sets the encryption key by encrypting it beforehand with the encryption key in the environment
func (k *keySetter) SetEncryptionKey(key []byte) error {
	var safes []Safe
	if err := getAll(k.db, "safe", &safes); err != nil {
		return err
	}
	if len(safes) != 0 {
		return fmt.Errorf("Encryption key is already set")
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
	return create(k.db, "safe", safe)
}
