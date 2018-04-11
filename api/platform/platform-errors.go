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

package platform

import (
	"fmt"

	"github.com/Peripli/service-manager/rest"
	"github.com/Sirupsen/logrus"
)

func externalError(err error, message string, args ...interface{}) error {
	if err != nil {
		logrus.Debug(err.Error())
	}
	return fmt.Errorf(message, args...)
}

func internalError(err error, message string, args ...interface{}) error {
	logrus.Error(err.Error())
	return fmt.Errorf(message, args...)
}

func errorSavePlatform(reason error) error {
	return internalError(reason, "could not save platform")
}

func errorPlatformLookup(reason error) error {
	return internalError(reason, "error occurred during platform lookup")
}

func errorMissingPlatform(platformID string) error {
	return externalError(nil, "could not find platform with id %s", platformID)
}

func errorRequestBodyDecode(reason error) error {
	return externalError(reason, "error occurred while decoding request body")
}

func responseProcessingError(err error) error {
	return internalError(err, "error while processing response")
}

func registerPlatformError(err error, statusCode int) *rest.ErrorResponse {
	return rest.CreateErrorResponse(err, statusCode, "RegisterPlatformError")
}

func getPlatformError(err error, statusCode int) *rest.ErrorResponse {
	return rest.CreateErrorResponse(err, statusCode, "GetPlatformError")
}

func getAllPlatformsError(err error, statusCode int) *rest.ErrorResponse {
	return rest.CreateErrorResponse(err, statusCode, "GetAllPlatformsError")
}

func deletePlatformError(err error, statusCode int) *rest.ErrorResponse {
	return rest.CreateErrorResponse(err, statusCode, "DeletePlatformError")
}

func updatePlatformError(err error, statusCode int) *rest.ErrorResponse {
	return rest.CreateErrorResponse(err, statusCode, "UpdatePlatformError")
}
