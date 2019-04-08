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
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/api/filters/authn/basic"
	"github.com/Peripli/service-manager/api/filters/authn/oauth"
	"github.com/Peripli/service-manager/api/info"
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/security"
	secfilters "github.com/Peripli/service-manager/pkg/security/filters"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

// Settings type to be loaded from the environment
type Settings struct {
	TokenIssuerURL    string `mapstructure:"token_issuer_url"`
	ClientID          string `mapstructure:"client_id"`
	SkipSSLValidation bool   `mapstructure:"skip_ssl_validation"`
	TokenBasicAuth    bool   `mapstructure:"token_basic_auth"`
}

// DefaultSettings returns default values for API settings
func DefaultSettings() *Settings {
	return &Settings{
		TokenIssuerURL:    "",
		ClientID:          "",
		SkipSSLValidation: false,
		TokenBasicAuth:    true, // RFC 6749 section 2.3.1
	}
}

// Validate validates the API settings
func (s *Settings) Validate() error {
	if (len(s.TokenIssuerURL)) == 0 {
		return fmt.Errorf("validate Settings: APITokenIssuerURL missing")
	}
	return nil
}

// NewInterceptableTransactionalRepository returns the minimum set of REST APIs needed for the Service Manager
func New(ctx context.Context, repository storage.Repository, settings *Settings, encrypter security.Encrypter) (*web.API, error) {
	bearerAuthnFilter, err := oauth.NewFilter(ctx, settings.TokenIssuerURL, settings.ClientID)
	if err != nil {
		return nil, err
	}

	return &web.API{
		// Default controllers - more filters can be registered using the relevant API methods
		Controllers: []web.Controller{
			NewController(repository, web.ServiceBrokersURL, types.ServiceBrokerType, func() types.Object {
				return &types.ServiceBroker{}
			}),
			NewController(repository, web.PlatformsURL, types.PlatformType, func() types.Object {
				return &types.Platform{}
			}),
			NewController(repository, web.VisibilitiesURL, types.VisibilityType, func() types.Object {
				return &types.Visibility{}
			}),
			NewServiceOfferingController(repository),
			NewServicePlanController(repository),
			&info.Controller{
				TokenIssuer:    settings.TokenIssuerURL,
				TokenBasicAuth: settings.TokenBasicAuth,
			},
			osb.NewController(&osb.StorageBrokerFetcher{
				BrokerStorage: repository,
			}, &osb.StorageCatalogFetcher{
				Repository: repository,
			},
				http.DefaultTransport,
			),
		},
		// Default filters - more filters can be registered using the relevant API methods
		Filters: []web.Filter{
			&filters.Logging{},
			basic.NewFilter(repository.Credentials(), encrypter),
			bearerAuthnFilter,
			secfilters.NewRequiredAuthnFilter(),
			&filters.SelectionCriteria{},
			&filters.PlatformAwareVisibilityFilter{},
		},
		Registry: health.NewDefaultRegistry(),
	}, nil
}
