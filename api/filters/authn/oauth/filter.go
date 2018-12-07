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

package oauth

import (
	"context"

	"github.com/Peripli/service-manager/pkg/security/middlewares"
	"github.com/Peripli/service-manager/pkg/web"
)

// BearerAuthnFilterName is the name of the bearer authentication filter
const BearerAuthnFilterName string = "BearerAuthnFilter"

// bearerAuthnFilter performs Bearer authentication by validating the Authorization header
type bearerAuthnFilter struct {
	web.Filter
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (ba *bearerAuthnFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(
					web.BrokersURL+"/**",
					web.PlatformsURL+"/**",
					web.ServiceOfferingsURL+"/**",
					web.ServicePlansURL+"/**",
					web.VisibilitiesURL+"/**",
				),
			},
		},
	}
}

// NewFilter returns a web.Filter for Bearer authentication
func NewFilter(ctx context.Context, tokenIssuer, clientID string) (web.Filter, error) {
	authenticator, err := newAuthenticator(ctx, &options{
		IssuerURL: tokenIssuer,
		ClientID:  clientID,
	})
	if err != nil {
		return nil, err
	}
	return &bearerAuthnFilter{
		Filter: middlewares.NewAuthnMiddleware(BearerAuthnFilterName, authenticator),
	}, nil
}
