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
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

type middleware struct {
	FilterName string
}

// Name implements the web.Filter interface and returns the identifier of the filter
func (m *middleware) Name() string {
	return m.FilterName
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (m *middleware) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{}
}

// ErrUserNotFound error when authentication passed but no user found
var ErrUserNotFound = errors.New("user identity must be provided when allowing authentication")

// UnauthorizedHTTPError returns HTTPError 401 with description from given error
func UnauthorizedHTTPError(err error) *util.HTTPError {
	return httpError(err, "authentication failed", "Unauthorized", http.StatusUnauthorized)
}

// ForbiddenHTTPError returns HTTPError 403
func ForbiddenHTTPError(err error) *util.HTTPError {
	return httpError(err, "authorization failed", "Forbidden", http.StatusForbidden)
}

func httpError(err error, description string, errorType string, statusCode int) *util.HTTPError {
	if err != nil {
		description = err.Error()
	}
	return &util.HTTPError{
		ErrorType:   errorType,
		Description: description,
		StatusCode:  statusCode,
	}
}
