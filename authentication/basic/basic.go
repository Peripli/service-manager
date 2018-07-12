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

package basic

import (
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/authentication"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
)

// Authenticator for basic authentication
type Authenticator struct {
	CredentialStorage storage.Credentials
}

// NewAuthenticator constructs a Basic authentication Authenticator
func NewAuthenticator(storage storage.Credentials) authentication.Authenticator {
	return &Authenticator{CredentialStorage: storage}
}

// Authenticate authenticates by using the provided Basic credentials
func (a *Authenticator) Authenticate(request *http.Request) (*authentication.User, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, web.NewHTTPError(
			errors.New("Missing or invalid Authorization header"),
			http.StatusUnauthorized,
			"Unauthorized")
	}

	credentials, err := a.CredentialStorage.Get(username)

	responseError := web.NewHTTPError(
		errors.New("Authentication failed"),
		http.StatusUnauthorized,
		"Unauthorized")
	if err == storage.ErrNotFound {
		logrus.Debugf("Username not found")
		return nil, responseError
	} else if credentials.Basic.Password != password {
		logrus.Debugf("Password mismatch")
		return nil, responseError
	}

	if err != nil {
		logrus.Errorf("Could not get credentials entity from storage")
		return nil, web.NewHTTPError(
			errors.New("Internal Server Error"),
			http.StatusInternalServerError,
			"InternalServerError")
	}

	return &authentication.User{
		Name: username,
	}, nil
}
