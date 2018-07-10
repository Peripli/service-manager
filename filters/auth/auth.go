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

package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Peripli/service-manager/authentication/basic"
	"github.com/Peripli/service-manager/authentication/oidc"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

const (
	basicScheme = "Basic"

	bearerScheme = "Bearer"
)

func (authFilter AuthenticationFilter) filterDispatcher(req *web.Request, handler web.Handler) (*web.Response, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return nil, web.NewHTTPError(
			errors.New("Missing Authorization header"),
			http.StatusUnauthorized,
			"Unauthorized")
	}

	header := strings.Split(authHeader, " ")
	scheme := header[0]

	switch scheme {
	case basicScheme:
		return authFilter.basicAuth(req, handler)
	case bearerScheme:
		return authFilter.oAuth(req, handler)
	}

	return nil, web.NewHTTPError(
		errors.New("Unsupported authentication scheme"),
		http.StatusUnauthorized,
		"Unauthorized")
}

func (authFilter AuthenticationFilter) basicAuth(req *web.Request, handler web.Handler) (*web.Response, error) {
	authenticator := basic.NewAuthenticator(authFilter.CredentialsStorage)
	_, err := authenticator.Authenticate(req.Request)
	if err != nil {
		return nil, err
	}

	return handler(req)
}

func (authFilter AuthenticationFilter) oAuth(req *web.Request, handler web.Handler) (*web.Response, error) {
	authenticator, err := oidc.NewAuthenticator(req.Request.Context(), oidc.Options{
		IssuerURL: authFilter.TokenIssuerURL,
		ClientID:  "smctl",
	})
	if err != nil {
		logrus.Error(err)
		return nil, web.NewHTTPError(
			errors.New("Authentication failed"),
			http.StatusBadRequest,
			"BadRequest")
	}

	_, err = authenticator.Authenticate(req.Request)
	if err != nil {
		return nil, web.NewHTTPError(
			errors.New("Authentication failed"),
			http.StatusUnauthorized,
			"Unauthorized")
	}

	return handler(req)
}
