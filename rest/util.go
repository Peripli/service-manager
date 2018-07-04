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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

// SendJSON writes a JSON value and sets the specified HTTP Status code
func SendJSON(writer http.ResponseWriter, code int, value interface{}) error {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	return encoder.Encode(value)
}

// NewJSONResponse turns plain object into a byte array representing JSON value and wraps it in web.Response
func NewJSONResponse(code int, value interface{}) (*web.Response, error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	body, err := json.Marshal(value)
	return &web.Response{
		StatusCode: code,
		Header:     headers,
		Body:       body,
	}, err
}

// ReadJSONBody parse request body
func ReadJSONBody(request *web.Request, value interface{}) error {
	err := json.Unmarshal(request.Body, value)
	if err != nil {
		logrus.Debug(err)
		return web.NewHTTPError(errors.New("Failed to decode request body"), http.StatusBadRequest, "BadRequest")
	}
	return nil
}
