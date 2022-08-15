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

package filters

import (
	"context"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/filters"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/authenticators"
)

// BearerAuthnFilterName is the name of the bearer authentication filter
const BearerAuthnFilterName string = "BearerAuthnFilter"

// NewOIDCAuthnFilter returns a web.Filter for Bearer authentication
func NewOIDCAuthnFilter(ctx context.Context, tokenIssuer, clientID string) (*filters.AuthenticationFilter, error) {
	authenticator, _, err := authenticators.NewOIDCAuthenticator(ctx, &authenticators.OIDCOptions{
		IssuerURL: tokenIssuer,
		ClientID:  clientID,
	})
	if err != nil {
		return nil, err
	}

	return filters.NewAuthenticationFilter(authenticator, BearerAuthnFilterName, oidcAuthnMatchers()), nil
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func oidcAuthnMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(
					web.ServiceBrokersURL+"/**",
					web.PlatformsURL+"/**",
					web.ServiceOfferingsURL+"/**",
					web.ServicePlansURL+"/**",
					web.VisibilitiesURL+"/**",
					web.ServiceInstancesURL+"/**",
					web.ConfigURL+"/**",
					web.ProfileURL+"/**",
					web.OperationsURL+"/**",
				),
			},
		},
	}
}
