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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/web"
)

// RequiredAuthorizationFilterName is the name of RequiredAuthorizationFilter
const RequiredAuthorizationFilterName = "RequiredAuthorizationFilter"

// requiredAuthzFilter type verifies that authorization has been performed for APIs that are secured
type requiredAuthzFilter struct{}

// NewRequiredAuthzFilter returns web.Filter which requires at least one authorization flows to be successful
func NewRequiredAuthzFilter() web.Filter {
	return &requiredAuthzFilter{}
}

// Name implements the web.Filter interface and returns the identifier of the filter
func (raf *requiredAuthzFilter) Name() string {
	return RequiredAuthorizationFilterName
}

// Run implements web.Filter and represents the authorization middleware function that verifies the user is
// authenticated
func (raf *requiredAuthzFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	if !web.IsAuthorized(ctx) {
		log.C(ctx).Info("No authorization confirmation found during execution of filter ", raf.Name())
		return nil, security.ForbiddenHTTPError("Not authorized")
	}
	return next.Handle(request)
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (raf *requiredAuthzFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(
					web.BrokersURL+"/**",
					web.PlatformsURL+"/**",
					web.OSBURL+"/**",
					web.ServicePlansURL+"/**",
					web.ServiceOfferingsURL+"/**",
					web.VisibilitiesURL+"/**",
				),
			},
		},
	}
}
