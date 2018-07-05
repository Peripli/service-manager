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
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/authentication/oidc"
	"github.com/Peripli/service-manager/authentication/basic"
	"errors"
	"strings"
	"fmt"
)


func (authFilter AuthenticationFilter) filterDispatcher(req *web.Request, handler web.Handler) (*web.Response, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("Missing Authorization header!")
	}

	header := strings.Split(authHeader, " ")
	schema := header[0]

	switch schema {
	case "Basic":
		fmt.Println("BASIC")
		return authFilter.basicAuth(req, handler)
	case "Bearer":
		fmt.Println("BEARER")
		return authFilter.oAuth(req, handler)
	}


	return &web.Response{
		StatusCode: 400,
	}, nil
}

// Filter which authenticates requests coming from Broker proxies using Basic Authentication
func (authFilter AuthenticationFilter) basicAuth(req *web.Request, handler web.Handler) (*web.Response, error) {
	authenticator := basic.NewAuthenticator(authFilter.CredentialsStorage)
	_, err := authenticator.Authenticate(req.Request)
	if err != nil {
		return nil, err
	}

	return handler(req)
}

// Filter which authenticates requests coming from Service Manger CLI using OAuth
func (authFilter AuthenticationFilter) oAuth(req *web.Request, handler web.Handler) (*web.Response, error) {
	authenticator, err := oidc.NewAuthenticator(req.Request.Context(), oidc.Options{
		IssuerURL: authFilter.TokenIssuerURL,
		ClientID: "cf",
	})
	if err != nil {
		return nil, err
	}

	_, err = authenticator.Authenticate(req.Request)
	if err != nil {
		return nil, err
	}

	return handler(req)
}