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

package authenticators

import (
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"golang.org/x/crypto/bcrypt"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

//BasicAuthenticatorFunc defines a function which attempts to authenticate a basic auth request
type BasicAuthenticatorFunc func(request *web.Request, repository storage.Repository, username, password string) (*web.UserContext, httpsec.Decision, error)

// Basic for basic security
type Basic struct {
	Repository             storage.Repository
	BasicAuthenticatorFunc BasicAuthenticatorFunc
}

// Authenticate authenticates by using the provided Basic credentials
func (a *Basic) Authenticate(request *web.Request) (*web.UserContext, httpsec.Decision, error) {
	username, password, ok := request.BasicAuth()
	if !ok {
		return nil, httpsec.Abstain, nil
	}

	return a.BasicAuthenticatorFunc(request, a.Repository, username, password)
}

//BasicPlatformAuthenticator attempts to authenticate basic auth requests with provided platform credentials
func BasicPlatformAuthenticator(request *web.Request, repository storage.Repository, username, password string) (*web.UserContext, httpsec.Decision, error) {
	ctx := request.Context()
	byUsername := query.ByField(query.EqualsOperator, "username", username)
	platformList, err := repository.List(ctx, types.PlatformType, byUsername)
	if err != nil {
		return nil, httpsec.Abstain, fmt.Errorf("could not get credentials entity from storage: %s", err)
	}

	if platformList.Len() != 1 {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	platform := platformList.ItemAt(0).(*types.Platform)
	if platform.Credentials.Basic.Password != password {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	return buildResponse(username, platform)
}

//BasicOSBAuthenticator attempts to authenticate basic auth requests with provided broker platform credentials
func BasicOSBAuthenticator(request *web.Request, repository storage.Repository, username, password string) (*web.UserContext, httpsec.Decision, error) {
	ctx := request.Context()

	brokerID, ok := request.PathParams[osb.BrokerIDPathParam]
	if !ok {
		return nil, httpsec.Abstain, fmt.Errorf("could not get authenticate OSB request: brokerID path parameter not found")
	}

	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", brokerID)
	byUsername := query.ByField(query.EqualsOperator, "username", username)

	credentialsList, err := repository.List(ctx, types.BrokerPlatformCredentialType, byBrokerID, byUsername)
	if err != nil {
		return nil, httpsec.Abstain, fmt.Errorf("could not get credentials entity from storage: %s", err)
	}

	useOldCredentials := false
	if credentialsList.Len() != 1 {
		byUsername.LeftOp = "old_username"
		credentialsList, err = repository.List(ctx, types.BrokerPlatformCredentialType, byBrokerID, byUsername)
		if err != nil {
			return nil, httpsec.Abstain, fmt.Errorf("could not get credentials entity from storage: %s", err)
		}

		if credentialsList.Len() != 1 {
			return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
		}

		useOldCredentials = true
	}

	credentials := credentialsList.ItemAt(0).(*types.BrokerPlatformCredential)

	passwordHash := credentials.PasswordHash
	if useOldCredentials {
		passwordHash = credentials.OldPasswordHash
	}

	if err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, httpsec.Deny, fmt.Errorf("provided credentials are invalid")
	}

	return buildResponse(username, credentials)
}

func buildResponse(username string, userData interface{}) (*web.UserContext, httpsec.Decision, error) {
	bytes, err := json.Marshal(userData)
	if err != nil {
		return nil, httpsec.Abstain, err
	}

	return &web.UserContext{
		Data: func(v interface{}) error {
			return json.Unmarshal(bytes, v)
		},
		AuthenticationType: web.Basic,
		Name:               username,
		AccessLevel:        web.NoAccess,
	}, httpsec.Allow, nil
}
