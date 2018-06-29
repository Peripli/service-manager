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

// Package sm contains logic for setting up the service manager server
package sm

import (
	"context"
	"errors"
	"fmt"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/log"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	security2 "github.com/Peripli/service-manager/storage/postgres/security"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// New creates a SM server
func New(ctx context.Context, cfg *config.Settings) (*server.Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	log.SetupLogging(cfg.Log)

	storage, err := storage.Use(ctx, postgres.Storage, cfg.Storage.URI)
	if err != nil {
		return nil, fmt.Errorf("error using storage: %v", err)
	}

	keyGetter := security2.NewKeyGetter(cfg.Security)
	encryptionKey, err := keyGetter.GetEncryptionKey()
	if err != nil {
		return nil, err
	}
	if len(encryptionKey) == 0 {
		logrus.Debug("No encryption key is present. Generating new one...")
		uuids, err := uuid.NewV4()
		if err != nil {
			logrus.Error(err)
			return nil, errors.New("Could not generate new encryption key " + err.Error())
		}
		newEncryptionKey := uuids.Bytes()
		keySetter := storage.KeySetter([]byte(cfg.Security.EncryptionKey))
		if err := keySetter.SetEncryptionKey(newEncryptionKey); err != nil {
			logrus.Error(err)
			return nil, err
		}
	}

	// TODO: this should remain API, but a service layer should be introduced between the API and the DB
	api := api.New(storage, cfg.API, cfg.Security)

	srv, err := server.New(api, cfg.Server)
	if err != nil {
		return nil, fmt.Errorf("error creating server: %v", err)
	}
	return srv, nil
}
