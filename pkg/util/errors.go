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

// HTTPError is an error type that provides error details that Service Manager error handlers would propagate to the client
type HTTPError struct {
	ErrorType   string `json:"error,omitempty"`
	Description string `json:"description,omitempty"`
	StatusCode  int    `json:"-"`
}

// Error HTTPError should implement error
func (e *HTTPError) Error() string {
	return e.Description
}

// UnsupportedQueryError is an error to show that the provided query cannot be executed
type UnsupportedQueryError struct {
	Message string
}

func (uq *UnsupportedQueryError) Error() string {
	return uq.Message
}

// WriteError sends a JSON containing the error to the response writer
func WriteError(err error, writer http.ResponseWriter) {
	var respError *HTTPError
	logger := log.D()
	switch t := err.(type) {
	case *HTTPError:
		logger.Errorf("HTTPError: %s", err)
		respError = t
	default:
		logger.Errorf("Unexpected error: %s", err)
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
	body, err := BodyToBytes(response.Body)
	if err != nil {
		return fmt.Errorf("error processing response body of resp with status code %d: %s", response.StatusCode, err)
	}

	err = fmt.Errorf("StatusCode: %d Body: %s", response.StatusCode, body)
	if response.Request != nil {
		log.C(response.Request.Context()).Errorf("Call to client failed with: %s", err)
	} else {
		log.D().Errorf("Call to client failed with: %s", err)
	}
	return err
}

var (
	// ErrNotFoundInStorage error returned from storage when entity is not found
	ErrNotFoundInStorage = errors.New("not found")

	// ErrAlreadyExistsInStorage error returned from storage when entity has conflicting fields
	ErrAlreadyExistsInStorage = errors.New("unique constraint violation")
)

type ErrBadRequestStorage error

// HandleStorageError handles storage errors by converting them to relevant HTTPErrors
func HandleStorageError(err error, entityName string) error {
	if err == nil {
		return nil
	}
	if entityName == "" {
		entityName = "entity"
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
			Description: fmt.Sprintf("could not find such %s", entityName),
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

func HandleSelectionError(err error, entityName ...string) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(*UnsupportedQueryError); ok {
		return &HTTPError{
			Description: err.Error(),
			ErrorType:   "BadRequest",
			StatusCode:  http.StatusBadRequest,
		}
	}
	if len(entityName) == 0 {
		entityName = []string{"entity"}
	}
	return HandleStorageError(err, entityName[0])
}
