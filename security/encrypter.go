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

type Encrypter struct{
	EncryptionGetter
}

func (e *Encrypter) Encrypt(plaintext []byte) ([]byte, error){
	key, err := e.EncryptionGetter.GetEncryptionKey()
	if err != nil {
		return nil, err
	}
	return Encrypt(plaintext, key)
}

func (e *Encrypter) Decrypt(ciphertext []byte) ([]byte, error){
	key, err := e.EncryptionGetter.GetEncryptionKey()
	if err != nil {
		return nil, err
	}
	return Decrypt(ciphertext, key)
}
