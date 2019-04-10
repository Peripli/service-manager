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
)

// Encrypter provides functionality to encrypt and decrypt data
//go:generate counterfeiter . Encrypter
type Encrypter interface {
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

// KeyFetcher provides functionality to get encryption key from a remote location
//go:generate counterfeiter . KeyFetcher
type KeyFetcher interface {
	GetEncryptionKey(ctx context.Context) ([]byte, error)
}

// KeySetter provides functionality to set encryption key in a remote location
//go:generate counterfeiter . KeySetter
type KeySetter interface {
	SetEncryptionKey(ctx context.Context, key []byte) error
}
