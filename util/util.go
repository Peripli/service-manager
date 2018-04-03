/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package util

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

// SendJSON writes a JSON value and sets the specified HTTP Status code
func SendJSON(writer http.ResponseWriter, code int, value interface{}) error {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	return encoder.Encode(value)
}

// GenerateCredentials return user and password
func GenerateCredentials() (string, string) {
	password := make([]byte, 128)
	user := make([]byte, 128)
	rand.Read(user)
	rand.Read(password)

	encodedPass := base64.StdEncoding.EncodeToString(password)
	encodedUser := base64.StdEncoding.EncodeToString(user)

	return encodedUser, encodedPass
}

// GenerateID return ID
func GenerateID() string {
	id := make([]byte, 64)
	rand.Read(id)
	encodedID := hex.EncodeToString(id)
	return encodedID
}
