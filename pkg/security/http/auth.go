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

package http

import (
	"context"
	"github.com/Peripli/service-manager/pkg/web"
)

// Decision represents a decision to allow or deny further
// processing or to abstain from taking a decision
type Decision int

var decisions = []string{"Abstain", "Allow", "Deny"}

const (
	// Abstain represents a decision to abstain from deciding - let another component decide
	Abstain Decision = iota

	// Allow represents decision to allow to proceed
	Allow

	// Deny represents decision to deny to proceed
	Deny
)

// String implements Stringer and converts the decision to human-readable value
func (a Decision) String() string {
	return decisions[a]
}

// Authenticator extracts the authenticator information from the request and
// returns information about the current user or an error if security was not successful
//go:generate counterfeiter . Authenticator
type Authenticator interface {
	// Authenticate returns information about the user if security is successful, a bool specifying
	// whether the authenticator ran or not and an error if one occurs
	Authenticate(req *web.Request) (*web.UserContext, Decision, error)
}

// Authorizer extracts the information from the authenticated user and
// returns a decision if the authorization passed
//go:generate counterfeiter . Authorizer
type Authorizer interface {
	// Authorize returns decision specifying whether the authorizer allowed, denied or abstained from giving access,
	// the access level associated with the decision and an error if one occurs
	Authorize(req *web.Request) (Decision, web.AccessLevel, error)
}

// TokenData represents the authentication token
//go:generate counterfeiter . TokenData
type TokenData interface {
	// Claims reads the claims from the token into the specified struct
	Claims(v interface{}) error
}

// TokenVerifier attempts to verify a token and returns it or an error if the verification was not successful
//go:generate counterfeiter . TokenVerifier
type TokenVerifier interface {
	// Verify verifies that the token is valid and returns a token if so, otherwise returns an error
	Verify(ctx context.Context, token string) (TokenData, error)
}
