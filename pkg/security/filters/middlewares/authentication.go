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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/web"
)

// Authentication type represents an authentication middleware
type Authentication struct {
	Authenticator http.Authenticator
}

// Run represents the authentication middleware function that delegates the authentication
// to the provided authenticator
func (m *Authentication) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	if _, ok := web.UserFromContext(ctx); ok {
		return next.Handle(request)
	}

	user, decision, err := m.Authenticator.Authenticate(request)
	if err != nil {
		if decision == http.Deny {
			log.C(ctx).Debug(err)
			request.Request = request.WithContext(web.ContextWithAuthenticationError(ctx, err))
			return next.Handle(request)
		}
		return nil, err
	}

	if decision == http.Allow {
		if user == nil {
			return nil, security.ErrUserNotFound
		}
		request.Request = request.WithContext(web.ContextWithUser(ctx, user))
	}

	return next.Handle(request)
}
