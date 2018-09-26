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

// TwoLayerEncrypter is an encrypter that fetches the encryption key from a remote location
type TwoLayerEncrypter struct {
	Fetcher KeyFetcher
}

// Encrypt encrypts the plaintext with a key obtained from a remote location
func (e *TwoLayerEncrypter) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	key, err := e.Fetcher.GetEncryptionKey(ctx)
	if err != nil {
		return nil, err
	}
	return Encrypt(plaintext, key)
}

// Decrypt decrypts the cipher text with a key obtained from a remote location
func (e *TwoLayerEncrypter) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	key, err := e.Fetcher.GetEncryptionKey(ctx)
	if err != nil {
		return nil, err
	}
	return Decrypt(ciphertext, key)
}
