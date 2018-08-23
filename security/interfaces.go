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

// Package security contains logic for setting SM security
package security

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

// AuthenticationDecision represents a Authenticator decision to allow or deny authentication or to abstain from
// taking a decision
type AuthenticationDecision int

var decisions = []string{"Allow", "Deny", "Abstain"}

const (
	// Allow represents an authentication decision to allow to proceed
	Allow AuthenticationDecision = iota

	// Deny represents an authentication decision to deny proceeding
	Deny

	// Abstain represents an authentication decision to abstain from deciding - let another component to decide
	Abstain
)

// String implements Stringer and converts the decision to human-readable value
func (a AuthenticationDecision) String() string {
	return decisions[a]
}

// Authenticator extracts the authenticator information from the request and
// returns information about the current user or an error if security was not successful
//go:generate counterfeiter . Authenticator
type Authenticator interface {
	// Authenticate returns information about the user if security is successful, a bool specifying
	// whether the authenticator ran or not and an error if one occurs
	Authenticate(req *http.Request) (*web.User, AuthenticationDecision, error)
}

// TokenVerifier attempts to verify a token and returns it or an error if the verification was not successful
//go:generate counterfeiter . TokenVerifier
type TokenVerifier interface {
	// Verify verifies that the token is valid and returns a token if so, otherwise returns an error
	Verify(ctx context.Context, token string) (web.TokenData, error)
}

// Encrypter provides functionality to encrypt and decrypt data
//go:generate counterfeiter . Encrypter
type Encrypter interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// KeyFetcher provides functionality to get encryption key from a remote location
//go:generate counterfeiter . KeyFetcher
type KeyFetcher interface {
	GetEncryptionKey() ([]byte, error)
}

// KeySetter provides functionality to set encryption key in a remote location
//go:generate counterfeiter . KeySetter
type KeySetter interface {
	SetEncryptionKey(key []byte) error
}
