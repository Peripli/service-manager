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

// Package authenticators contains logic for setting up an Open ID Connect authenticator
package authenticators

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"

	"fmt"

	"github.com/coreos/go-oidc"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	goidc "github.com/coreos/go-oidc"
)

type claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"user_name"`
	Email    string `json:"email"`
	Verified bool   `json:"email_verified"`
}

type providerJSON struct {
	Issuer  string `json:"issuer"`
	JWKSURL string `json:"jwks_uri"`
}

// OIDCOptions is the configuration used to construct a new OIDC authenticator
type OIDCOptions struct {
	// IssuerURL is the base URL of the token issuer
	IssuerURL string

	// ClientID is the id of the oauth client used to verify the tokens
	ClientID string

	// ReadConfigurationFunc is the function used to call the token issuer. If one is not provided, http.DefaultClient.Do will be used
	ReadConfigurationFunc util.DoRequestFunc
}

type oidcVerifier struct {
	*oidc.IDTokenVerifier
}

// Verify implements security.TokenVerifier and delegates to oidc.IDTokenVerifier
func (v *oidcVerifier) Verify(ctx context.Context, idToken string) (httpsec.TokenData, error) {
	return v.IDTokenVerifier.Verify(ctx, idToken)
}

// OauthAuthenticator is the OpenID implementation of security.Authenticator
type OauthAuthenticator struct {
	Verifier httpsec.TokenVerifier
}

// NewOIDCAuthenticator returns a new OpenID authenticator or an error if one couldn't be configured
func NewOIDCAuthenticator(ctx context.Context, options *OIDCOptions) (httpsec.Authenticator, string, error) {
	if options.IssuerURL == "" {
		return nil, "", errors.New("missing issuer URL")
	}
	resp, err := getOpenIDConfig(ctx, options)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", util.HandleResponseError(resp)
	}

	var p providerJSON
	if err = util.BodyToObject(resp.Body, &p); err != nil {
		return nil, "", fmt.Errorf("error decoding body of response with status %s: %s", resp.Status, err.Error())
	}

	keySet := goidc.NewRemoteKeySet(ctx, p.JWKSURL)
	return &OauthAuthenticator{Verifier: &oidcVerifier{
		IDTokenVerifier: goidc.NewVerifier(p.Issuer, keySet, newOIDCConfig(options)),
	}}, p.Issuer, nil
}

func newOIDCConfig(options *OIDCOptions) *goidc.Config {
	return &goidc.Config{
		ClientID:          options.ClientID,
		SkipClientIDCheck: options.ClientID == "",
	}
}

// Authenticate returns information about the user by obtaining it from the bearer token, or an error if security is unsuccessful
func (a *OauthAuthenticator) Authenticate(request *web.Request) (*web.UserContext, httpsec.Decision, error) {
	authorizationHeader := request.Header.Get("Authorization")
	if authorizationHeader == "" || !strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
		return nil, httpsec.Abstain, nil
	}
	if a.Verifier == nil {
		return nil, httpsec.Abstain, errors.New("authenticator is not configured")
	}
	token := strings.TrimSpace(authorizationHeader[len("Bearer "):])
	if token == "" {
		return nil, httpsec.Deny, nil
	}
	idToken, err := a.Verifier.Verify(request.Context(), token)
	if err != nil {
		return nil, httpsec.Deny, err
	}
	claims := &claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, httpsec.Deny, err
	}
	return &web.UserContext{
		Data:               idToken.Claims,
		AuthenticationType: web.Bearer,
		Name:               claims.Username,
		AccessLevel:        web.NoAccess,
	}, httpsec.Allow, nil
}

func getOpenIDConfig(ctx context.Context, options *OIDCOptions) (*http.Response, error) {
	// Work around for UAA until https://github.com/cloudfoundry/uaa/issues/805 is fixed
	// Then goidc.NewProvider(ctx, options.IssuerURL) should be used
	if _, err := url.ParseRequestURI(options.IssuerURL); err != nil {
		return nil, err
	}
	wellKnown := strings.TrimSuffix(options.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequest(http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, err
	}

	var readConfigFunc util.DoRequestFunc
	if options.ReadConfigurationFunc != nil {
		readConfigFunc = options.ReadConfigurationFunc
	} else {
		readConfigFunc = http.DefaultClient.Do
	}

	resp, err := readConfigFunc(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return resp, err
}
