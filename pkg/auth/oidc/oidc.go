/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"net/http"

	"github.com/Peripli/service-manager/pkg/auth"
	"github.com/Peripli/service-manager/pkg/auth/util"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type openIDConfiguration struct {
	TokenEndpoint         string              `json:"token_endpoint"`
	AuthorizationEndpoint string              `json:"authorization_endpoint"`
	MTLSEndpointAliases   MTLSEndpointAliases `json:"mtls_endpoint_aliases"`
}

type MTLSEndpointAliases struct {
	TokenEndpoint         string `json:"token_endpoint"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
}

// OpenIDStrategy implementation of OpenID strategy
type OpenIDStrategy struct {
	oauth2Config *oauth2.Config
	ccConfig     *clientcredentials.Config
	httpClient   *http.Client
}

// NewOpenIDStrategy returns OpenId auth strategy
func NewOpenIDStrategy(options *auth.Options) (*OpenIDStrategy, *auth.Options, error) {
	var httpClient *http.Client
	var err error

	httpClient, err = util.BuildHTTPClient(options)
	if err != nil {
		return nil, nil, err
	}

	httpClient.Timeout = options.Timeout

	var oauthConfig *oauth2.Config
	var ccConfig *clientcredentials.Config

	openIDConfig, err := fetchOpenidConfiguration(options.IssuerURL, httpClient.Do)
	if err != nil {
		return nil, nil, fmt.Errorf("error occurred while fetching openid configuration: %s", err)
	}

	options.AuthorizationEndpoint, options.TokenEndpoint = RetrieveAuthEndpoints(openIDConfig, util.MtlsEnabled(options))

	oauthConfig = newOauth2Config(options)

	ccConfig = newClientCredentialsConfig(options)

	return &OpenIDStrategy{
		oauth2Config: oauthConfig,
		ccConfig:     ccConfig,
		httpClient:   httpClient,
	}, options, nil
}

func RetrieveAuthEndpoints(openIDConfig *openIDConfiguration, mtlsEnabled bool) (string, string) {
	var AuthorizationEndpoint string
	var TokenEndpoint string
	if mtlsEnabled {
		AuthorizationEndpoint = openIDConfig.MTLSEndpointAliases.AuthorizationEndpoint
		TokenEndpoint = openIDConfig.MTLSEndpointAliases.TokenEndpoint
	} else {
		AuthorizationEndpoint = openIDConfig.AuthorizationEndpoint
		TokenEndpoint = openIDConfig.TokenEndpoint
	}
	return AuthorizationEndpoint, TokenEndpoint
}

// ClientCredentials is used to perform client credentials grant type flow
func (s *OpenIDStrategy) ClientCredentials() (*auth.Token, error) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, s.httpClient)
	token, err := s.ccConfig.Token(ctx)

	if err != nil {
		log.D().Debugf("authenticator: %s", err.Error())
		return nil, wrapError(err)
	}

	resultToken := &auth.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.Expiry,
		TokenType:    token.TokenType,
	}

	return resultToken, err
}

// PasswordCredentials is used to perform password grant type flow
func (s *OpenIDStrategy) PasswordCredentials(user, password string) (*auth.Token, error) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, s.httpClient)
	token, err := s.oauth2Config.PasswordCredentialsToken(ctx, user, password)
	if err != nil {
		return nil, wrapError(err)
	}

	resultToken := &auth.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.Expiry,
		TokenType:    token.TokenType,
	}

	return resultToken, err
}

func wrapError(err error) error {
	oauth2Err, ok := err.(*oauth2.RetrieveError)
	log.D().Debugf("oidc error: %s", oauth2Err)
	if ok {
		type A struct {
			Description string `json:"error_description"`
		}
		a := A{}
		unmarshalErr := json.Unmarshal(oauth2Err.Body, &a)
		if unmarshalErr != nil {
			a.Description = string(oauth2Err.Body)
		}

		return fmt.Errorf("auth error: %s", a.Description)
	}
	return err
}
