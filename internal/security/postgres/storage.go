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

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/security"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

const schema = "vault"

type storage struct {
	db            *sqlx.DB
	encryptionKey []byte
}

// NewSecureStorage returns a security storage for obtaining encryption keys from a database
func NewSecureStorage(ctx context.Context, securitySettings api.Security) (security.Storage, error) {
	if securitySettings.URI == "" {
		return nil, fmt.Errorf("storage URI cannot be empty")
	}
	db, err := sqlx.Connect("postgres", securitySettings.URI)
	if err != nil {
		return nil, err
	}
	go awaitTermination(ctx, db)
	return &storage{db, []byte(securitySettings.EncryptionKey)}, nil
}

func awaitTermination(ctx context.Context, db *sqlx.DB) {
	<-ctx.Done()
	logrus.Debug("Context cancelled. Closing storage...")
	if err := db.Close(); err != nil {
		logrus.Error(err)
	}
}

// Fetcher returns a KeyFetcher configured to fetch a key from the database
func (s *storage) Fetcher() security.KeyFetcher {
	return &keyFetcher{s.db, []byte(s.encryptionKey)}
}

// Setter returns a KeySetter configured to set a key in the database
func (s *storage) Setter() security.KeySetter {
	return &keySetter{s.db, s.encryptionKey}
}
