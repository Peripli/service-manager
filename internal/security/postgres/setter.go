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

type keySetter struct {
	db            *sqlx.DB
	encryptionKey []byte
}

// Sets the encryption key by encrypting it beforehand with the encryption key in the environment
func (k *keySetter) SetEncryptionKey(key []byte) error {
	bytes, err := security.Encrypt(key, k.encryptionKey)
	if err != nil {
		return err
	}
	safe := security.Safe{
		Secret:    string(bytes),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tx, err := k.db.Beginx()
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			if txError := tx.Rollback(); txError != nil {
				logrus.Error("Could not rollback Transaction", txError)
			}
		}
	}()

	if _, err = tx.NamedExec(fmt.Sprintf("INSERT INTO %s.safe (secret, created_at, updated_at)"+
		" VALUES(:secret, :created_at, :updated_at)", schema), &safe); err == nil {
		if err := tx.Commit(); err == nil {
			ok = true
		} else {
			return err
		}
	}
	return err
}
