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

	"fmt"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
)

// Authenticator for basic security
type Authenticator struct {
	CredentialStorage      storage.Credentials
	CredentialsTransformer security.CredentialsTransformer
}

// NewAuthenticator constructs a Basic authentication Authenticator
func NewAuthenticator(storage storage.Credentials, transformer security.CredentialsTransformer) security.Authenticator {
	return &Authenticator{CredentialStorage: storage, CredentialsTransformer: transformer}
}

// Authenticate authenticates by using the provided Basic credentials
func (a *Authenticator) Authenticate(request *http.Request) (*security.User, security.AuthenticationDecision, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, security.Abstain, nil
	}

	credentials, err := a.CredentialStorage.Get(username)

	passwordBytes, err := a.CredentialsTransformer.Reverse([]byte(credentials.Basic.Password))
	if err != nil {
		return nil, security.Deny, err
	}

	if err == util.ErrNotFoundInStorage || string(passwordBytes) != password {
		return nil, security.Deny, nil
	}

	if err != nil {
		return nil, security.Abstain, fmt.Errorf("could not get credentials entity from storage: %s", err)
	}

	return &security.User{
		Name: username,
	}, security.Allow, nil
}
