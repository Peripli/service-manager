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
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

const authFilterName = "AuthenticationFilter"

type AuthenticationFilter struct {
	CredentialsStorage storage.Credentials
	TokenIssuerURL string
}

func NewAuthenticationFilter(credentialStorage storage.Credentials, tokenIssuerURL string) AuthenticationFilter {
	return AuthenticationFilter{
		CredentialsStorage: credentialStorage,
		TokenIssuerURL: tokenIssuerURL,
	}
}

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
				Methods: []string{http.MethodGet},
				PathPattern: "/v1/service_brokers/**",
			},
			Middleware: authFilter.filterDispatcher,
		},
		{
			Name: authFilterName,
			RouteMatcher: web.RouteMatcher{
				Methods: []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
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