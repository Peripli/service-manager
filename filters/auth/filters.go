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

package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/authentication/basic"
	"github.com/Peripli/service-manager/config"

	"github.com/Peripli/service-manager/authentication"

	"github.com/Peripli/service-manager/authentication/oidc"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const authFilterName = "AuthenticationFilter"

// AuthenticationFilter holds authentication information
type AuthenticationFilter struct {
	oAuthAuthenticator authentication.Authenticator
	basicAuthenticator authentication.Authenticator
}

// NewAuthenticationFilter constructs a new AuthenticationFilter object
func NewAuthenticationFilter(ctx context.Context, credentialStorage storage.Credentials, cfg *config.Settings) AuthenticationFilter {
	oauthAuthenticator, err := oidc.NewAuthenticator(ctx, oidc.Options{
		ClientID:  cfg.OAuth.ClientID,
		IssuerURL: cfg.API.TokenIssuerURL,
	})
	if err != nil {
		panic(fmt.Errorf("Could not construct OAuth authenticator. Reason: %s", err))
	}

	basicAuthenticator := basic.NewAuthenticator(credentialStorage)

	return AuthenticationFilter{
		oAuthAuthenticator: oauthAuthenticator,
		basicAuthenticator: basicAuthenticator,
	}
}

// Filters returns the default authentication filters
func (authFilter AuthenticationFilter) Filters() []web.Filter {
	return []web.Filter{
		{
			Name: authFilterName,
			RouteMatcher: web.RouteMatcher{
				PathPattern: "/v1/osb/**",
			},
			Middleware: authFilter.basicAuth,
		},
		{
			Name: authFilterName,
			RouteMatcher: web.RouteMatcher{
				Methods:     []string{http.MethodGet},
				PathPattern: "/v1/service_brokers/**",
			},
			Middleware: authFilter.filterDispatcher,
		},
		{
			Name: authFilterName,
			RouteMatcher: web.RouteMatcher{
				Methods:     []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
				PathPattern: "/v1/service_brokers/**",
			},
			Middleware: authFilter.oAuth,
		},
		{
			Name: authFilterName,
			RouteMatcher: web.RouteMatcher{
				PathPattern: "/v1/platforms/**",
			},
			Middleware: authFilter.oAuth,
		},
		{
			Name: authFilterName,
			RouteMatcher: web.RouteMatcher{
				PathPattern: "/v1/sm_catalog",
			},
			Middleware: authFilter.oAuth,
		},
	}
}
