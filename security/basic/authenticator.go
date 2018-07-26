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

// Package basic contains logic for setting up a basic authenticator
package basic

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
)

// Authenticator for basic security
type Authenticator struct {
	CredentialStorage storage.Credentials
}

// NewAuthenticator constructs a Basic security Authenticator
func NewAuthenticator(storage storage.Credentials) *Authenticator {
	return &Authenticator{CredentialStorage: storage}
}

// Authenticate authenticates by using the provided Basic credentials
func (a *Authenticator) Authenticate(request *http.Request) (*security.User, bool, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, false, nil
	}

	credentials, err := a.CredentialStorage.Get(username)

	responseError := &util.HTTPError{
		ErrorType:   "Unauthorized",
		Description: "authentication failed",
		StatusCode:  http.StatusUnauthorized,
	}
	if err == storage.ErrNotFound {
		return nil, true, responseError
	}

	if err != nil {
		logrus.Errorf("Could not get credentials entity from storage")
		return nil, true, &util.HTTPError{
			ErrorType:   "InternalServerError",
			Description: "Internal Server Error",
			StatusCode:  http.StatusInternalServerError,
		}
	}

	if credentials.Basic.Password != password {
		return nil, true, responseError
	}

	return &security.User{
		Name: username,
	}, true, nil
}
