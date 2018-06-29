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

package security

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/security"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type getter struct {
	db            *sqlx.DB
	encryptionkey []byte
}

func NewKeyFetcher(ctx context.Context, settings security.Settings) *getter {
	db, err := sqlx.Connect("postgres", settings.URI)
	if err != nil {
		logrus.Panicln("Could not connect to PostgreSQL secure storage: ", err)
	}
	go awaitTermination(ctx, db)
	return &getter{db, []byte(settings.EncryptionKey)}
}

func awaitTermination(ctx context.Context, db *sqlx.DB) {
	<-ctx.Done()
	logrus.Debug("Context cancelled. Closing storage...")
	if err := db.Close(); err != nil {
		logrus.Error(err)
	}
}

// GetEncryptionKey returns the encryption key used to encrypt the credentials for brokers
func (s *getter) GetEncryptionKey() ([]byte, error) {
	var safes []Safe
	if err := s.db.Select(&safes, fmt.Sprintf("SELECT * FROM %s.safe", schema)); err != nil {
		return nil, err
	}
	if len(safes) != 1 {
		logrus.Warnf("Unexpected number of keys found: %d", len(safes))
		return []byte{}, nil
	}
	encryptedKey := []byte(safes[0].Secret)
	return security.Decrypt(encryptedKey, s.encryptionkey)
}
