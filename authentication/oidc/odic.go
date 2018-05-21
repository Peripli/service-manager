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

package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Peripli/service-manager/authentication"
	"github.com/coreos/go-oidc"
)

// Options is the configuration used to construct a new OIDC authenticator
type Options struct {
	IssuerURL string
	ClientID  string
	Client    *http.Client
}

// Authenticator is the OIDC implementation of authentication.Authenticator
type Authenticator struct {
	Verifier authentication.TokenVerifier
}

// NewAuthenticator returns a new OIDC authenticator or an error if one couldn't be configured
func NewAuthenticator(ctx context.Context, options Options) (authentication.Authenticator, error) {
	// Work around for UAA until https://github.com/cloudfoundry/uaa/issues/805 is fixed
	// Then oidc.NewProvider(ctx, options.IssuerURL) should be used
	wellKnown := strings.TrimSuffix(options.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequest("GET", wellKnown, nil)
	if err != nil {
		return nil, err
	}

	var client *http.Client
	if options.Client == nil {
		client = options.Client
	} else {
		client = http.DefaultClient
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", resp.Status, body)
	}

	var p providerJSON
	err = json.Unmarshal(body, &p)
	if err != nil {
		return nil, err
	}

	keySet := oidc.NewRemoteKeySet(ctx, p.JWKSURL)
	cfg := &oidc.Config{
		ClientID: options.ClientID,
	}
	return &Authenticator{Verifier: &goOidcVerifier{oidc.NewVerifier(p.Issuer, keySet, cfg)}}, nil
}

func (a *Authenticator) Authenticate(request *http.Request) (*authentication.User, error) {
	authorizationHeader := request.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("Missing authorization header")
	}
	token := strings.TrimPrefix(authorizationHeader, "Bearer ")
	if token == "" {
		return nil, errors.New("Token is required in authorization header")
	}
	idToken, err := a.Verifier.Verify(request.Context(), token)
	if err != nil {
		return nil, err
	}
	claims := &claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, err
	}
	return &authentication.User{
		Name: claims.Username,
		UID:  claims.UserID,
	}, nil
}
