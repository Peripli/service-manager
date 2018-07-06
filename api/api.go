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
	"github.com/Peripli/service-manager/api/broker"
	"github.com/Peripli/service-manager/api/catalog"
	"github.com/Peripli/service-manager/api/info"
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/api/platform"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

// Settings type to be loaded from the environment
type Settings struct {
	TokenIssuerURL string `mapstructure:"token_issuer_url"`
}

// New returns the minimum set of REST APIs needed for the Service Manager
func New(storage storage.Storage, settings Settings) *rest.API {
	return &rest.API{
		Controllers: []rest.Controller{
			&broker.Controller{
				BrokerStorage:       storage.Broker(),
				OSBClientCreateFunc: osbc.NewClient,
			},
			&platform.Controller{
				PlatformStorage: storage.Platform(),
			},
			&info.Controller{
				TokenIssuer: settings.TokenIssuerURL,
			},
			&catalog.Controller{
				BrokerStorage: storage.Broker(),
			},
			&osb.Controller{
				BrokerStorage: storage.Broker(),
			},
			&platform.Controller{
				PlatformStorage: storage.Platform(),
			},
		},
	}
}
