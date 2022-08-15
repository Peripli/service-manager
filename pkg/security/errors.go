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

package security

import (
	"errors"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

// ErrUserNotFound error when authentication passed but no user found
var ErrUserNotFound = errors.New("user identity must be provided when allowing authentication")

// UnauthorizedHTTPError returns HTTPError 401 with some description
func UnauthorizedHTTPError(description string) error {
	return &util.HTTPError{
		ErrorType:   "Unauthorized",
		Description: description,
		StatusCode:  http.StatusUnauthorized,
	}
}

// ForbiddenHTTPError returns HTTPError 403 with some description
func ForbiddenHTTPError(description string) error {
	return &util.HTTPError{
		ErrorType:   "Forbidden",
		Description: description,
		StatusCode:  http.StatusForbidden,
	}
}
