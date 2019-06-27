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

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/ws"

	apiNotifications "github.com/Peripli/service-manager/api/notifications"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/api/info"
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/health"
	secfilters "github.com/Peripli/service-manager/pkg/security/filters"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const osbVersion = "2.13"

// Settings type to be loaded from the environment
type Settings struct {
	TokenIssuerURL    string   `mapstructure:"token_issuer_url" description:"url of the token issuer which to use for validating tokens"`
	ClientID          string   `mapstructure:"client_id" description:"id of the client from which the token must be issued"`
	SkipSSLValidation bool     `mapstructure:"skip_ssl_validation" description:"whether to skip ssl verification when making calls to external services"`
	TokenBasicAuth    bool     `mapstructure:"token_basic_auth" description:"specifies if client credentials to the authorization server should be sent in the header as basic auth (true) or in the body (false)"`
	ProctedLabels     []string `mapstructure:"protected_labels" description:"defines labels which cannot be modified/added by REST API requests"`
	OSBVersion        string   `mapstructure:"-"`
}

// DefaultSettings returns default values for API settings
func DefaultSettings() *Settings {
	return &Settings{
		TokenIssuerURL:    "",
		ClientID:          "",
		SkipSSLValidation: false,
		TokenBasicAuth:    true, // RFC 6749 section 2.3.1
		OSBVersion:        osbVersion,
	}
}

// Validate validates the API settings
func (s *Settings) Validate() error {
	if (len(s.TokenIssuerURL)) == 0 {
		return fmt.Errorf("validate Settings: APITokenIssuerURL missing")
	}
	return nil
}

type Options struct {
	Repository  storage.Repository
	APISettings *Settings
	WSSettings  *ws.Settings
	Notificator storage.Notificator
}

// New returns the minimum set of REST APIs needed for the Service Manager
func New(ctx context.Context, options *Options) (*web.API, error) {
	bearerAuthnFilter, err := filters.NewOIDCAuthnFilter(ctx, options.APISettings.TokenIssuerURL, options.APISettings.ClientID)
	if err != nil {
		return nil, err
	}

	return &web.API{
		// Default controllers - more filters can be registered using the relevant API methods
		Controllers: []web.Controller{
			NewController(options.Repository, web.ServiceBrokersURL, types.ServiceBrokerType, func() types.Object {
				return &types.ServiceBroker{}
			}),
			NewController(options.Repository, web.PlatformsURL, types.PlatformType, func() types.Object {
				return &types.Platform{}
			}),
			NewController(options.Repository, web.VisibilitiesURL, types.VisibilityType, func() types.Object {
				return &types.Visibility{}
			}),
			apiNotifications.NewController(ctx, options.Repository, options.WSSettings, options.Notificator),
			NewServiceOfferingController(options.Repository),
			NewServicePlanController(options.Repository),
			&info.Controller{
				TokenIssuer:    options.APISettings.TokenIssuerURL,
				TokenBasicAuth: options.APISettings.TokenBasicAuth,
			},
			&osb.Controller{
				BrokerFetcher: func(ctx context.Context, brokerID string) (*types.ServiceBroker, error) {
					byID := query.ByField(query.EqualsOperator, "id", brokerID)
					br, err := options.Repository.Get(ctx, types.ServiceBrokerType, byID)
					if err != nil {
						return nil, util.HandleStorageError(err, "broker")
					}
					return br.(*types.ServiceBroker), nil
				},
			},
		},
		// Default filters - more filters can be registered using the relevant API methods
		Filters: []web.Filter{
			&filters.Logging{},
			filters.NewBasicAuthnFilter(options.Repository),
			bearerAuthnFilter,
			secfilters.NewRequiredAuthnFilter(),
			&filters.SelectionCriteria{},
			filters.NewProtectedLabelsFilter(options.APISettings.ProctedLabels),
			&filters.PlatformAwareVisibilityFilter{},
			&filters.PatchOnlyLabelsFilter{},
		},
		Registry: health.NewDefaultRegistry(),
	}, nil
}
