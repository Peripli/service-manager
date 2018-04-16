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

package rest

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

// ErrorResponse struct used to store information about error
type ErrorResponse struct {
	Error       string `json:"error,omitempty"`
	Description string `json:"description"`
}

// HandleError sends a JSON containing the error to the response writer
func HandleError(err error, writer http.ResponseWriter) {
	if err != nil {
		sendErr := SendJSON(writer, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
		if sendErr != nil {
			logrus.Errorf("Could not write error to response: %v", sendErr)
		}
	}
}

// SendJSON writes a JSON value and sets the specified HTTP Status code
func SendJSON(writer http.ResponseWriter, code int, value interface{}) error {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	return encoder.Encode(value)
}
