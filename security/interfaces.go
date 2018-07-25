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

package security

import (
	"context"
	"net/http"
)

// User holds the information for the current user
type User struct {
	Name string `json:"name"`
}

// BasicAuthenticator extracts the authenticator information from the request and
// returns information about the current user or an error if security was not successful
//go:generate counterfeiter . Authenticator
type Authenticator interface {
	// Authenticate returns information about the user if security is successful, a bool specifying
	// whether the authenticator ran or not and an error if one occurs
	Authenticate(req *http.Request) (*User, bool, error)
}

// Token interface provides means to unmarshal the claims in a struct
//go:generate counterfeiter . Token
type Token interface {
	// Claims unmarshals the claims into the specified struct
	Claims(v interface{}) error
}

// TokenVerifier attempts to verify a token and returns it or an error if the verification was not successful
//go:generate counterfeiter . TokenVerifier
type TokenVerifier interface {
	// Verify verifies that the token is valid and returns a token if so, otherwise returns an error
	Verify(ctx context.Context, token string) (Token, error)
}
