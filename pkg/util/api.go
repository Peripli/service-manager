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

// Package util contains web utils for APIs, clients and error handling
package util

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

var (
	reservedSymbolsRFC3986 = strings.Join([]string{
		":", "/", "?", "#", "[", "]", "@", "!", "$", "&", "'", "(", ")", "*", "+", ",", ";", "=",
	}, "")
)

// EmptyResponseBody represents an empty response body value
type EmptyResponseBody struct{}

// InputValidator should be implemented by types that need input validation check. For a reference refer to pkg/types
type InputValidator interface {
	Validate() error
}

// HasRFC3986ReservedSymbols returns true if input contains any reserver characters as defined in RFC3986 section oidc_authn.oidc_authn
func HasRFC3986ReservedSymbols(input string) bool {
	return strings.ContainsAny(input, reservedSymbolsRFC3986)
}

// ToRFCFormat converts a time.Time timestamp to RFC3339 format
func ToRFCFormat(timestamp time.Time) string {
	return timestamp.UTC().Format(time.RFC3339)
}

// RequestBodyToBytes reads the request body and returns []byte with its content or an error if
// the media type is incorrect or if the body is not a valid JSON
func RequestBodyToBytes(request *http.Request) ([]byte, error) {
	contentType := request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, &HTTPError{
			ErrorType:   "InvalidMediaType",
			Description: "invalid media type provided",
			StatusCode:  http.StatusUnsupportedMediaType,
		}
	}

	body, err := BodyToBytes(request.Body)
	if err != nil {
		return nil, err
	}

	if !json.Valid(body) {
		return nil, &HTTPError{
			ErrorType:   "BadRequest",
			Description: "request body is not valid JSON",
			StatusCode:  http.StatusBadRequest,
		}
	}

	return body, nil
}

// BytesToObject converts the provided bytes to object and validates it
func BytesToObject(bytes []byte, object interface{}) error {
	if err := unmarshal(bytes, object); err != nil {
		return err
	}
	if err := validate(object); err != nil {
		return err
	}

	return nil
}

// unmarshal unmarshals the specified []byte into the provided value and returns an HttpError in unmarshaling fails
func unmarshal(body []byte, value interface{}) error {
	err := json.Unmarshal(body, value)
	if err != nil {
		log.D().Error("Failed to decode request body: ", err)
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: "Failed to decode request body",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}

// validate validates the specified value in case it implements InputValidator
func validate(value interface{}) error {
	if input, ok := value.(InputValidator); ok {
		if err := input.Validate(); err != nil {
			return &HTTPError{
				ErrorType:   "BadRequest",
				Description: input.Validate().Error(),
				StatusCode:  http.StatusBadRequest,
			}
		}
	}
	return nil
}

// WriteJSON writes a JSON value and sets the specified HTTP Status code
func WriteJSON(writer http.ResponseWriter, code int, value interface{}) error {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	return encoder.Encode(value)
}

// NewJSONResponse turns plain object into a byte array representing JSON value and wraps it in web.Response
func NewJSONResponse(code int, value interface{}) (*web.Response, error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	body := make([]byte, 0)
	var err error
	if _, ok := value.(EmptyResponseBody); !ok {
		body, err = json.Marshal(value)
	}

	return &web.Response{
		StatusCode: code,
		Header:     headers,
		Body:       body,
	}, err
}
