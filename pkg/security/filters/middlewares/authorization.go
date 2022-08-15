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
	"fmt"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// Authorization type represents an authorization middleware
type Authorization struct {
	Authorizer http.Authorizer
}

// Run represents the authorization middleware function that delegates the authorization
// to the provided authorizer
func (m *Authorization) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	if web.IsAuthorized(ctx) {
		return next.Handle(request)
	}

	decision, accessLevel, err := m.Authorizer.Authorize(request)
	if err != nil {
		if decision == http.Deny {
			log.C(ctx).Debug(err)
			request.Request = request.WithContext(web.ContextWithAuthorizationError(ctx, err))
			return next.Handle(request)
		}
		return nil, err
	}

	if decision == http.Allow {
		userContext, found := web.UserFromContext(ctx)
		if !found {
			return nil, fmt.Errorf("authorization failed due to missing user context")
		}
		userContext.AccessLevel = accessLevel
		request.Request = request.WithContext(web.ContextWithUser(ctx, userContext))
		if accessLevel == web.NoAccess {
			return nil, fmt.Errorf("authorization failed due to missing access level. Authorizer that allows access should also specify the access level")
		}
		if !web.IsAuthorized(ctx) {
			request.Request = request.WithContext(web.ContextWithAuthorization(ctx))
		}
	}

	return next.Handle(request)
}
