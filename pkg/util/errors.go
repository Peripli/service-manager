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

	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// HTTPError is an error type that provides error details compliant with the Open Service Broker API conventions
type HTTPError struct {
	ErrorType   string `json:"error,omitempty"`
	Description string `json:"description,omitempty"`
	StatusCode  int    `json:"-"`
}

// Error HTTPError should implement error
func (e *HTTPError) Error() string {
	return e.Description
}

// WriteError sends a JSON containing the error to the response writer
func WriteError(err error, writer http.ResponseWriter) {
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

	sendErr := WriteJSON(writer, respError.StatusCode, respError)
	if sendErr != nil {
		logrus.Errorf("Could not write error to response: %v", sendErr)
	}
}

// HandleResponseError builds at HttpErrorResponse from the given response.
func HandleResponseError(response *http.Response) error {
	logrus.Errorf("Handling failure response: returned status code %d", response.StatusCode)
	httpErr := &HTTPError{
		StatusCode: response.StatusCode,
	}

	bytes, err := BodyToBytes(response.Body)
	if err != nil {
		return err
	}

	r := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &r); err != nil {
		httpErr.Description = fmt.Errorf("could not read error response: %s", err).Error()
		return httpErr
	}

	httpErr.ErrorType = cast.ToString(r["error"])
	httpErr.Description = cast.ToString(r["description"])
	if httpErr.Description == "" {
		httpErr.Description = string(bytes)
	}
	return httpErr
}
