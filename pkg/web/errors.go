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

package web

// HTTPError struct used to store information about error
type HTTPError struct {
	ErrorType   string `json:"error,omitempty"`
	Description string `json:"description"`
	StatusCode  int    `json:"-"`
}

// Error HTTPError should implement error
func (errorResponse *HTTPError) Error() string {
	return errorResponse.Description
}

// NewHTTPError creates HTTPError object
func NewHTTPError(err error, statusCode int, errorType string) *HTTPError {
	return &HTTPError{
		ErrorType:   errorType,
		Description: err.Error(),
		StatusCode:  statusCode}
}
