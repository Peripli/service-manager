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
	"net/url"
	"strings"

	"github.com/Peripli/service-manager/authentication"
	"github.com/coreos/go-oidc"
	"github.com/sirupsen/logrus"
	"crypto/tls"
)

// Options is the configuration used to construct a new OIDC authenticator
type Options struct {
	// IssuerURL is the base URL of the token issuer
	IssuerURL string

	// ClientID is the id of the oauth client used to verify the tokens
	ClientID string

	// ReadConfigurationFunc is the function used to call the token issuer. If one is not provided, http.DefaultClient.Do will be used
	ReadConfigurationFunc DoRequestFunc
}

// DoRequestFunc is an alias for any function that takes an http request and returns a response and error
type DoRequestFunc func(request *http.Request) (*http.Response, error)

// Authenticator is the OpenID implementation of authentication.Authenticator
type Authenticator struct {
	Verifier authentication.TokenVerifier
}

// NewAuthenticator returns a new OpenID authenticator or an error if one couldn't be configured
func NewAuthenticator(ctx context.Context, options Options) (authentication.Authenticator, error) {
	// Work around for UAA until https://github.com/cloudfoundry/uaa/issues/805 is fixed
	// Then oidc.NewProvider(ctx, options.IssuerURL) should be used
	if _, err := url.ParseRequestURI(options.IssuerURL); err != nil {
		return nil, err
	}
	wellKnown := strings.TrimSuffix(options.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequest(http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, err
	}

	var readConfigFunc DoRequestFunc
	if options.ReadConfigurationFunc != nil {
		readConfigFunc = options.ReadConfigurationFunc
	} else {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		readConfigFunc = http.DefaultClient.Do
	}

	resp, err := readConfigFunc(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.Errorf("OpenID configuration response body couldn't be closed", err)
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code %s: %s", resp.Status, body)
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
	return &Authenticator{Verifier: &oidcVerifier{oidc.NewVerifier(p.Issuer, keySet, cfg)}}, nil
}

// Authenticate returns information about the user by obtaining it from the bearer token, or an error if authentication is unsuccessful
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
	}, nil
}
