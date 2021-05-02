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
	"context"
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
func WriteError(ctx context.Context, err error, writer http.ResponseWriter) {
	logger := log.C(ctx)
	respError := ToHTTPError(ctx, err)
	sendErr := WriteJSON(writer, respError.StatusCode, respError)
	if sendErr != nil {
		logger.Errorf("Could not write error to response: %v", sendErr)
	}
}

func ToHTTPError(ctx context.Context, err error) *HTTPError {
	var respError *HTTPError
	logger := log.C(ctx)
	switch t := err.(type) {
	case *UnsupportedQueryError:
		logger.Errorf("UnsupportedQueryError: %s", err)
		respError = &HTTPError{
			ErrorType:   "BadRequest",
			Description: err.Error(),
			StatusCode:  http.StatusBadRequest,
		}
	case *HTTPError:
		// if status code was not set, default to internal server error
		if t.StatusCode == 0 {
			t.StatusCode = http.StatusInternalServerError
		}
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

	return respError
}

// HandleResponseError builds an error from the given response
func HandleResponseError(response *http.Response) error {
	body, err := BodyToBytes(response.Body)
	if err != nil {
		body = []byte(fmt.Sprintf("error reading response body: %s", err))
	}

	err = fmt.Errorf("StatusCode: %d Body: %s", response.StatusCode, body)
	if response.Request != nil {
		return fmt.Errorf("request %s %s failed: %s", response.Request.Method, response.Request.URL, err)
	}
	return fmt.Errorf("request failed: %s", err)
}

var (
	// ErrNotFoundInStorage error returned from storage when entity is not found
	ErrNotFoundInStorage = errors.New("not found")

	// ErrSharedInstanceHasReferences error returned when shared instance has references
	ErrSharedInstanceHasReferences = errors.New("shared instance has references")

	// ErrAlreadyExistsInStorage error returned from storage when entity has conflicting fields
	ErrAlreadyExistsInStorage = errors.New("unique constraint violation")

	// ErrConcurrentResourceModification error returned when concurrent resource updates are happening
	ErrConcurrentResourceModification = errors.New("another resource update happened concurrently. Please reattempt the update")

	// ErrInvalidNotificationRevision provided notification revision is not valid, must return http status GONE
	ErrInvalidNotificationRevision = errors.New("notification revision is not valid")
)

// ErrBadRequestStorage represents a storage error that should be translated to http.StatusBadRequest
type ErrBadRequestStorage struct {
	Cause error
}

func (e *ErrBadRequestStorage) Error() string {
	return e.Cause.Error()
}

// ErrForeignKeyViolation represents a foreign key constraint storage error that should be translated to a user-friendly http.StatusBadRequest
type ErrForeignKeyViolation struct {
	Entity          string
	ReferenceEntity string
}

func (e *ErrForeignKeyViolation) Error() string {
	return fmt.Sprintf("Could not delete entity %s due to existing reference entity %s", e.Entity, e.ReferenceEntity)
}

// HandleStorageError handles storage errors by converting them to relevant HTTPErrors
func HandleStorageError(err error, entityName string) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(*HTTPError); ok {
		return err
	}

	if len(entityName) == 0 {
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
	case ErrConcurrentResourceModification:
		return &HTTPError{
			ErrorType:   "ConcurrentResourceUpdate",
			Description: "Another concurrent resource update occurred. Please reattempt the update operation",
			StatusCode:  http.StatusPreconditionFailed,
		}
	default:
		// in case we did not replace the pg.Error in the DB layer, propagate it as response message to give the caller relevant info
		switch e := err.(type) {
		case *ErrForeignKeyViolation:
			return &HTTPError{
				ErrorType:   "ExistingReferenceEntity",
				Description: fmt.Sprintf("Error deleting entity %s: %s", entityName, err.Error()),
				StatusCode:  http.StatusConflict,
			}
		case *ErrBadRequestStorage:
			return &HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("storage err: %s", e.Error()),
				StatusCode:  http.StatusBadRequest,
			}
		default:
			return err
		}
	}
}

func HandleReferencesError(err error, guidsArray []string) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(*HTTPError); ok {
		return err
	}

	switch err {
	case ErrSharedInstanceHasReferences:
		errorMessage := fmt.Sprintf("Couldn't delete the service instance. Before you can delete it, you first need to delete these %d references: %s", len(guidsArray), guidsArray)
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: errorMessage,
			StatusCode:  http.StatusBadRequest,
		}
	default:
		return err
	}
}

var (
	ErrCatalogUsesReservedPlanName           = errors.New("catalog contains a reserved plan name")
	ErrPlanMustBeBindable                    = errors.New("plan must be bindable")
	ErrReferencedInstanceNotShared           = errors.New("referenced-instance should be shared first")
	ErrChangingPlanOfReferenceInstance       = errors.New("changing plan of reference instance")
	ErrChangingPlanOfSharedInstance          = errors.New("changing plan of shared instance")
	ErrChangingParametersOfReferenceInstance = errors.New("changing parameters of reference instance")
	ErrMissingReferenceParameter             = errors.New("missing referenced_instance_id parameter")
	ErrParsingNewCatalogWithReference        = errors.New("failed generating reference-plan")
	ErrUnknownOSBMethod                      = errors.New("osb method is unknown")
	ErrSharedPlanHasReferences               = errors.New("shared plan has references")
	ErrInstanceIsAlreadyAtDesiredSharedState = errors.New("instance already at the desired shared state")
	ErrInvalidShareRequest                   = errors.New("invalid share request")
)

func HandleInstanceSharingError(err error, entityName string) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(*HTTPError); ok {
		return err
	}

	switch err {
	case ErrCatalogUsesReservedPlanName:
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("The plan name you used for registration \"%s\" is reserved for the Service Manager; you must choose a different name.", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrPlanMustBeBindable:
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("The plan %s must be defined as 'bindable' so that it can support instance sharing.", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrReferencedInstanceNotShared:
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Failed to create the reference. The instance %s, for which you want to create the reference, must be shared first.", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrChangingPlanOfReferenceInstance:
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Failed to update the instance %s. This is a reference instance, therefore its plan can't be changed.", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrChangingPlanOfSharedInstance:
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Couldn't update the instance's plan. The instance %s is shared, therefore you must un-share it first.", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrChangingParametersOfReferenceInstance:
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Failed to update the instance %s. This is a reference instance, therefore its parameters can't be changed.", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrMissingReferenceParameter:
		return &HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("Failed to create the instance. Missing parameter \"%s\".", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrParsingNewCatalogWithReference:
		return &HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("Parsing failed. Couldn't generate the new catalog with the plan \"%s\".", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrUnknownOSBMethod:
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Unknown OSB method: \"%s\".", entityName),
			StatusCode:  http.StatusBadRequest,
		}
	case ErrSharedPlanHasReferences:
		errorMessage := fmt.Sprintf("Couldn't update the service plan. Before you can set it as supportInstanceSharing=false, you first need to un-share the instances of the plan: %s.", entityName)
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: errorMessage,
			StatusCode:  http.StatusBadRequest,
		}
	case ErrInvalidShareRequest:
		errorMessage := fmt.Sprintf("could not set the 'shared' property of the instance %s with other changes at the same time.", entityName)
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: errorMessage,
			StatusCode:  http.StatusBadRequest,
		}
	case ErrInstanceIsAlreadyAtDesiredSharedState:
		errorMessage := fmt.Sprintf("the service instance %s, is already at the desried \"shared\" state", entityName)
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: errorMessage,
			StatusCode:  http.StatusBadRequest,
		}
	default:
		return err
	}
}
