/*
 * Copyright 2018 The Service Manager Authors
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

// Package api contains logic for building the Service Manager API business logic
package api

import (
	"fmt"

	"context"

	"github.com/Peripli/service-manager/api/broker"
	"github.com/Peripli/service-manager/api/catalog"
	"github.com/Peripli/service-manager/api/filters/authn"
	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/api/info"
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/api/platform"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

// Security is the configuration used for the encryption of data
type Security struct {
	// EncryptionKey is the encryption key from the environment
	EncryptionKey string `mapstructure:"encryption_key"`
}

// Validate validates the API Security settings
func (s *Security) Validate() error {
	if len(s.EncryptionKey) != 32 {
		return fmt.Errorf("validate Settings: SecurityEncryptionkey length must be exactly 32")
	}
	return nil
}

// Settings type to be loaded from the environment
type Settings struct {
	TokenIssuerURL string `mapstructure:"token_issuer_url"`
	ClientID       string `mapstructure:"client_id"`
	Security       Security `mapstructure:"security"`
}

// Validate validates the API settings
func (s *Settings) Validate() error {
	if (len(s.TokenIssuerURL)) == 0 {
		return fmt.Errorf("validate Settings: APITokenIssuerURL missing")
	}
	if (len(s.ClientID)) == 0 {
		return fmt.Errorf("validate Settings: APIClientID missing")
	}
	if err := s.Security.Validate(); err != nil {
		return err
	}
	return nil
}

// New returns the minimum set of REST APIs needed for the Service Manager
func New(ctx context.Context, storage storage.Storage, settings Settings, encrypter security.Encrypter) (*web.API, error) {
	bearerAuthnFilter, err := authn.NewBearerAuthnFilter(ctx, settings.TokenIssuerURL, settings.ClientID)
	if err != nil {
		return nil, err
	}
	return &web.API{
		// Default controllers - more filters can be registered using the relevant API methods
		Controllers: []web.Controller{
			&broker.Controller{
				BrokerStorage:       storage.Broker(),
				OSBClientCreateFunc: osbc.NewClient,
				Encrypter:           encrypter,
			},
			&platform.Controller{
				PlatformStorage:        storage.Platform(),
				Encrypter: encrypter,
			},
			&info.Controller{
				TokenIssuer: settings.TokenIssuerURL,
			},
			&catalog.Controller{
				BrokerStorage: storage.Broker(),
			},
			&osb.Controller{
				BrokerStorage:          storage.Broker(),
				Encrypter: encrypter,
			},
			&healthcheck.Controller{
				Storage: storage,
			},
		},
		// Default filters - more filters can be registered using the relevant API methods
		Filters: []web.Filter{
			authn.NewBasicAuthnFilter(storage.Credentials(), encrypter),
			bearerAuthnFilter,
			authn.NewRequiredAuthnFilter(),
		},
	}, nil
}
