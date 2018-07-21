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
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// HTTPError is an error type that provides error details compliant with the Open Service Broker API conventions
type HTTPError struct {
	ErrorType   string `json:"error,omitempty"`
	Description string `json:"description,omitempty"`
	StatusCode  int    `json:"-"`
}

//TODO put error types here

// Error HTTPError should implement error
func (e *HTTPError) Error() string {
	return e.Description
}

// HandleError sends a JSON containing the error to the response writer
func HandleAPIError(err error, writer http.ResponseWriter) {
	var respError *HTTPError
	switch t := err.(type) {
	case *HTTPError:
		logrus.Debug(err)
		respError = t
	default:
		logrus.Error(err)
		respError = &HTTPError{
			ErrorType:   "InternalError",
			Description: "Internal server error",
			StatusCode:  http.StatusInternalServerError,
		}
	}

	sendErr := SendJSON(writer, respError.StatusCode, respError)
	if sendErr != nil {
		logrus.Errorf("Could not write error to response: %v", sendErr)
	}
}

// HandleClientResponseError builds at HttpErrorResponse from the given response.
func HandleClientResponseError(response *http.Response) error {
	logrus.Errorf("Handling failure response: returned status code %d", response.StatusCode)
	httpErr := &HTTPError{
		StatusCode: response.StatusCode,
	}

	if err := ReadClientResponseContent(httpErr, response.Body); err != nil {
		httpErr.Description = err.Error()
		return fmt.Errorf("error handling failure response: %s", err)
	}

	return httpErr
}
