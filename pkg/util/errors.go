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
	"errors"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
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
	logger := log.D()
	switch t := err.(type) {
	case *HTTPError:
		logger.Debug(err)
		respError = t
	default:
		logger.Error(err)
		respError = &HTTPError{
			ErrorType:   "InternalError",
			Description: "Internal server error",
			StatusCode:  http.StatusInternalServerError,
		}
	}

	sendErr := WriteJSON(writer, respError.StatusCode, respError)
	if sendErr != nil {
		logger.Errorf("Could not write error to response: %v", sendErr)
	}
}

// HandleResponseError builds at HttpErrorResponse from the given response.
func HandleResponseError(response *http.Response) error {
	logger := log.D()
	logger.Errorf("Handling failure response: returned status code %d", response.StatusCode)
	httpErr := &HTTPError{
		StatusCode: response.StatusCode,
	}

	body, err := BodyToBytes(response.Body)
	if err != nil {
		return fmt.Errorf("error processing response body of resp with status code %d: %s", response.StatusCode, err)
	}

	if err := BytesToObject(body, httpErr); err != nil || httpErr.Description == "" {
		logger.Debugf("Failure response with status code %d is not an HTTPError. Error converting body: %v. Default err will be returned.", response.StatusCode, err)
		return fmt.Errorf("StatusCode: %d Body: %s", response.StatusCode, body)
	}
	return httpErr
}

var (
	// ErrNotFoundInStorage error returned from storage when entity is not found
	ErrNotFoundInStorage = errors.New("not found")

	// ErrAlreadyExistsInStorage error returned from storage when entity has conflicting fields
	ErrAlreadyExistsInStorage = errors.New("unique constraint violation")
)

type ErrBadRequestStorage error

// HandleStorageError handles storage errors by converting them to relevant HTTPErrors
func HandleStorageError(err error, entityName, entityID string) error {
	if err == nil {
		return nil
	}
	switch err {
	case ErrAlreadyExistsInStorage:
		return &HTTPError{
			ErrorType:   "Conflict",
			Description: fmt.Sprintf("found conflicting %s", entityName),
			StatusCode:  http.StatusConflict,
		}
	case ErrNotFoundInStorage:
		return &HTTPError{
			ErrorType:   "NotFound",
			Description: fmt.Sprintf("could not find %s with id %s", entityName, entityID),
			StatusCode:  http.StatusNotFound,
		}
	default:
		// in case we did not replace the pg.Error in the DB layer, propagate it as response message to give the caller relevant info
		storageErr, ok := err.(ErrBadRequestStorage)
		if ok {
			return &HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("storage err: %s", storageErr.Error()),
				StatusCode:  http.StatusBadRequest,
			}
		}
	}
	return fmt.Errorf("unknown error type returned from storage layer: %s", err)
}
