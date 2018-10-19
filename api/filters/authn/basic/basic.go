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
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/security/middlewares"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

// BasicAuthnFilterName is the name of the basic authentication filter
const BasicAuthnFilterName string = "BasicAuthnFilter"

// NewFilter returns a web.Filter for basic auth using the provided
// credentials storage in order to validate the credentials
func NewFilter(storage storage.Credentials, encrypter security.Encrypter) web.Filter {
	return &basicAuthnFilter{
		Filter: middlewares.NewAuthnMiddleware(
			BasicAuthnFilterName,
			&basicAuthenticator{CredentialStorage: storage, Encrypter: encrypter},
		),
	}
}

// basicAuthnFilter performs Basic authentication by validating the Authorization header
type basicAuthnFilter struct {
	web.Filter
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (ba *basicAuthnFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.BrokersURL + "/**"),
			},
		},
	}
}

// basicAuthenticator for basic security
type basicAuthenticator struct {
	CredentialStorage storage.Credentials
	Encrypter         security.Encrypter
}

// Authenticate authenticates by using the provided Basic credentials
func (a *basicAuthenticator) Authenticate(request *http.Request) (*web.User, security.Decision, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, security.Abstain, nil
	}

	ctx := request.Context()
	credentials, err := a.CredentialStorage.Get(ctx, username)

	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return nil, security.Deny, err
		}
		return nil, security.Abstain, fmt.Errorf("could not get credentials entity from storage: %s", err)
	}
	passwordBytes, err := a.Encrypter.Decrypt(ctx, []byte(credentials.Basic.Password))
	if err != nil {
		return nil, security.Abstain, fmt.Errorf("could not reverse credentials from storage: %v", err)
	}

	if string(passwordBytes) != password {
		return nil, security.Deny, nil
	}

	return &web.User{
		Name: username,
	}, security.Allow, nil
}
