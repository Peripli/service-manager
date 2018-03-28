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
)

type ErrorResponse struct {
	Error       string `json:"error,omitempty"`
	Description string `json:"description"`
}

// ErrorHandlerFunc wraps an APIHandler and returns an http.Handler by providing a central error handling place for all APIHandlers
func ErrorHandlerFunc(handler APIHandler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, reader *http.Request) {
		err := handler(writer, reader)
		HandlerError(err, writer)
	})
}

// HandleError sends a JSON containing the error to the response writer
func HandlerError(err error, writer http.ResponseWriter) {
	if err != nil {
		SendJSON(writer, http.StatusInternalServerError, ErrorResponse{
			Description: err.Error(),
		})
	}
}

// SendJSON writes a JSON value and sets the specified HTTP Status code
func SendJSON(writer http.ResponseWriter, code int, value interface{}) error {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	return encoder.Encode(value)
}
