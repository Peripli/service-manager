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

// CredentialsTransformer provides functionality to modify credentials and to reverse already modified credentials
type CredentialsTransformer interface {
	Transform(secret []byte) ([]byte, error)
	Reverse(cipher []byte) ([]byte, error)
}

// Encrypter provides functionality to encrypt and decrypt data
type Encrypter interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// KeyFetcher provides functionality to get encryption key from a remote location
type KeyFetcher interface {
	GetEncryptionKey() ([]byte, error)
}

// KeySetter provides functionality to set encryption key in a remote location
type KeySetter interface {
	SetEncryptionKey(key []byte) error
}