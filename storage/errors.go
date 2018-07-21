/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version oidc_authn.0 (the "License");
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

package storage

import (
	"fmt"
	"net/http"

	"errors"

	"github.com/Peripli/service-manager/pkg/util"
)

var (
	// ErrNotFound error returned from storage when entity is not found
	ErrNotFound = errors.New("not found")

	// ErrUniqueViolation error returned from storage when entity has conflicting fields
	ErrUniqueViolation = errors.New("unique constraint violation")
)

// CheckErrors check multiple HTTPError errors
func CheckErrors(errors ...error) error {
	if len(errors) == 0 {
		return nil
	}
	for _, err := range errors {
		if err != nil {
			_, ok := err.(*util.HTTPError)
			if ok {
				return err
			}
		}
	}
	return errors[0]
}

// HandleNotFoundError checks if entity is not found
func HandleNotFoundError(err error, entityName string, entityID string) error {
	if err == ErrNotFound {
		return &util.HTTPError{
			ErrorType:   "NotFound",
			Description: fmt.Sprintf("could not find %s with id %s", entityName, entityID),
			StatusCode:  http.StatusNotFound,
		}
	}
	return err
}

// HandleUniqueError checks if entity
func HandleUniqueError(err error, entityName string) error {
	if err == ErrUniqueViolation {
		return &util.HTTPError{
			ErrorType:   "Conflict",
			Description: fmt.Sprintf("found conflicting %s", entityName),
			StatusCode:  http.StatusConflict,
		}
	}
	return err
}
