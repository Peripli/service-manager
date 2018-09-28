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

package middlewares

import (
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/web"
)

// NewAuthzMiddleware returns web.Filter which uses the given security.Authorizer
// to authorize the request. FilterMatchers should be extended to cover the desired
// endpoints.
func NewAuthzMiddleware(filterName string, authorizer security.Authorizer) web.Filter {
	return &authzMiddleware{
		middleware: &middleware{
			FilterName: filterName,
		},
		Authorizer: authorizer,
	}
}

// authzMiddleware type represents an authorization middleware
type authzMiddleware struct {
	*middleware
	Authorizer security.Authorizer
}

// Run represents the authorization middleware function that delegates the authorization
// to the provided authorizer
func (m *authzMiddleware) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	if web.GetAuthorizationConfirmation(ctx) {
		return next.Handle(request)
	}

	decision, err := m.Authorizer.Authorize(request.Request)
	if err != nil {
		if decision == security.Deny {
			return nil, ForbiddenHTTPError(err)
		}
		return nil, err
	}

	switch decision {
	case security.Allow:
		request.Request = request.WithContext(web.ConfirmAuthorization(ctx))
	case security.Deny:
		return nil, ForbiddenHTTPError(nil)
	}

	return next.Handle(request)
}
