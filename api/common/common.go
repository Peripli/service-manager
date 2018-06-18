/*
 *    Copyright 2018 The Service Manager Authors
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

package common

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/filter"

	"github.com/Peripli/service-manager/storage"
)

// CheckErrors check multiple ErrorResponse errors
func CheckErrors(errors ...error) error {
	if len(errors) == 0 {
		return nil
	}
	for _, err := range errors {
		if err != nil {
			_, ok := err.(*filter.ErrorResponse)
			if ok {
				return err
			}
		}
	}
	return errors[0]
}

// HandleNotFoundError checks if entity is not found
func HandleNotFoundError(err error, entityName string, entityID string) error {
	if err == storage.ErrNotFound {
		return filter.NewErrorResponse(
			fmt.Errorf("Could not find %s with id %s", entityName, entityID),
			http.StatusNotFound,
			"NotFound")
	}
	return err
}

// HandleUniqueError checks if entity
func HandleUniqueError(err error, entityName string) error {
	if err == storage.ErrUniqueViolation {
		return filter.NewErrorResponse(
			fmt.Errorf("Found conflicting %s", entityName),
			http.StatusConflict,
			"Conflict")
	}
	return err
}
