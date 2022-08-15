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
	"fmt"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// RequiredAuthenticationFilterName is the name of RequiredAuthenticationFilter
const RequiredAuthenticationFilterName = "RequiredAuthenticationFilter"

// NewRequiredAuthnFilter returns web.Filter
func NewRequiredAuthnFilter(matchers []web.FilterMatcher) web.Filter {
	return &requiredAuthnFilter{
		matchers: matchers,
	}
}

// requiredAuthnFilter type verifies that authentication has been performed for APIs that are secured
type requiredAuthnFilter struct {
	matchers []web.FilterMatcher
}

// Name implements the web.Filter interface and returns the identifier of the filter
func (raf *requiredAuthnFilter) Name() string {
	return RequiredAuthenticationFilterName
}

// Run implements web.Filter and represents the authentication middleware function that verifies the user is
// authenticated
func (raf *requiredAuthnFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	if _, ok := web.UserFromContext(ctx); !ok {
		log.C(ctx).Error("No authenticated user found in request context during execution of filter ", raf.Name())
		message := "No authenticated user found"
		if found, err := web.AuthenticationErrorFromContext(ctx); found {
			message = fmt.Sprintf("%s: %s", message, err)
		}
		return nil, security.UnauthorizedHTTPError(message)
	}

	return next.Handle(request)
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (raf *requiredAuthnFilter) FilterMatchers() []web.FilterMatcher {
	return raf.matchers
}
