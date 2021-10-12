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

package auth

import (
	"net/http"
	"time"
)

// Flow defiens an OAuth 2 authentication flow, a.k.a. grant type
type Flow string

const (
	// DefaultFlow is the default auth flow
	DefaultFlow Flow = ""
	// ClientCredentials flow used for technical user
	ClientCredentials Flow = "client-credentials"
	// PasswordGrant flow used for named users
	PasswordGrant Flow = "password-grant"
)

// Options is used to configure new authenticators and clients
type Options struct {
	User                  string
	Password              string
	ClientID              string `mapstructure:"client_id"`
	ClientSecret          string `mapstructure:"client_secret"`
	Certificate           string `mapstructure:"cert"`
	Key                   string `mapstructure:"key"`
	AuthorizationEndpoint string `mapstructure:"authorization_endpoint"`
	TokenEndpoint         string `mapstructure:"token_endpoint"`
	IssuerURL             string `mapstructure:"issuer_url"`
	AuthFlow              Flow   `mapstructure:"auth_flow"`

	TokenBasicAuth bool `mapstructure:"token_basic_auth"`
	SSLDisabled    bool `mapstructure:"ssl_disabled"`

	Timeout time.Duration `mapstructure:"timeout"`
}

// Token contains the structure of a typical UAA response token
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    time.Time `json:"expires_in"`
	Scope        string    `json:"scope"`
}

// Authenticator should be implemented for different authentication strategies
//go:generate counterfeiter . Authenticator
type Authenticator interface {
	ClientCredentials() (*Token, error)
	PasswordCredentials(user, password string) (*Token, error)
}

// Client should be implemented for http like clients which do automatic authentication
//go:generate counterfeiter . Client
type Client interface {
	Do(*http.Request) (*http.Response, error)
}
