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
	"net/http"

	"github.com/Peripli/service-manager/util"
	"github.com/Sirupsen/logrus"
)

// ErrorHandlerFunc wraps an APIHandler and returns an http.Handler by providing a central error handling place for all APIHandlers
func ErrorHandlerFunc(handler APIHandler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, reader *http.Request) {
		if err := handler(writer, reader); err != nil {
			handleError(err, writer)
		}
	})
}

// handleError sends a JSON containing the error to the response writer
func handleError(err error, writer http.ResponseWriter) {
	var respError *ErrorResponse
	switch t := err.(type) {
	case *ErrorResponse:
		respError = t
	case ErrorResponse:
		respError = &t
	default:
		respError = CreateErrorResponse(err, http.StatusInternalServerError, "GeneralError")
	}

	sendErr := util.SendJSON(writer, respError.StatusCode, respError)
	if sendErr != nil {
		logrus.Errorf("Could not write error to response: %v", sendErr)
	}
}

// CreateErrorResponse create ErrorResponse object
func CreateErrorResponse(err error, statusCode int, errorType string) *ErrorResponse {
	return &ErrorResponse{
		ErrorType:   errorType,
		Description: err.Error(),
		StatusCode:  statusCode}
}
