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
	"net/http"
	"github.com/Peripli/service-manager/authentication"
	"errors"
	"github.com/Peripli/service-manager/storage"
)

type Authenticator struct {
	CredentialStorage storage.Credentials
}

func NewAuthenticator(storage storage.Credentials) (authentication.Authenticator) {
	return 	&Authenticator{CredentialStorage: storage}
}

func (a *Authenticator) Authenticate(request *http.Request) (*authentication.User, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, errors.New("Missing or invalid Authorization header!")
	}

	credentials, err := a.CredentialStorage.Get(username)
	if err != nil || credentials.Basic.Password != password {
		return nil, err
	}

	return &authentication.User{
		Name: username,
	}, nil
}