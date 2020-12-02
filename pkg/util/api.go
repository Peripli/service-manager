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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/log"
)

const jsonContentType = "application/json"

var (
	reservedSymbolsRFC3986 = strings.Join([]string{
		":", "/", "?", "#", "[", "]", "@", "!", "$", "&", "'", "(", ")", "*", "+", ",", ";", "=",
	}, "")
	supportedContentTypes = []string{jsonContentType, "application/x-www-form-urlencoded"}
)

// InputValidator should be implemented by types that need input validation check. For a reference refer to pkg/types
type InputValidator interface {
	Validate() error
}

// HasRFC3986ReservedSymbols returns true if input contains any reserver characters as defined in RFC3986 section oidc_authn.oidc_authn
func HasRFC3986ReservedSymbols(input string) bool {
	return strings.ContainsAny(input, reservedSymbolsRFC3986)
}

// ToRFCNanoFormat converts a time.Time timestamp to RFC3339Nano format
func ToRFCNanoFormat(timestamp time.Time) string {
	return timestamp.UTC().Format(time.RFC3339Nano)
}

// RequestBodyToBytes reads the request body and returns []byte with its content or an error if
// the media type is unsupported or if the body is not a valid JSON
func RequestBodyToBytes(request *http.Request) ([]byte, error) {
	contentType, err := validateContentTypeIsSupported(request)
	if err != nil {
		return nil, err
	}

	body, err := BodyToBytes(request.Body)
	if err != nil {
		return nil, err
	}

	if contentType == jsonContentType {
		if err := validJson(body); err != nil {
			return nil, &HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("Request body is not valid: %s", err),
				StatusCode:  http.StatusBadRequest,
			}
		}
	}

	return body, nil
}

func validateContentTypeIsSupported(request *http.Request) (string, error) {
	contentTypeHeader := request.Header.Get("Content-Type")
	if len(contentTypeHeader) == 0 {
		request.Header.Set("Content-Type", jsonContentType)
		return jsonContentType, nil
	}

	mimeType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return "", &HTTPError{
			ErrorType:   "UnsupportedMediaType",
			Description: fmt.Sprintf("media type error: %s", err),
			StatusCode:  http.StatusUnsupportedMediaType,
		}
	}

	if slice.StringsAnyEquals(supportedContentTypes, mimeType) {
		return mimeType, nil
	}

	return "", &HTTPError{
		ErrorType:   "UnsupportedMediaType",
		Description: fmt.Sprintf("unsupported media type: %s", contentTypeHeader),
		StatusCode:  http.StatusUnsupportedMediaType,
	}
}

func validJson(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("invalid json")
	}
	return checkDuplicateKeys(json.NewDecoder(bytes.NewReader(data)), nil)
}

func checkDuplicateKeys(d *json.Decoder, path []string) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok { // There's nothing to do for simple values (strings, numbers, bool, nil)
		return nil
	}
	switch delim {
	case '{':
		keys := make(map[string]bool)
		for d.More() {
			t, err := d.Token() // Get the key
			if err != nil {
				return err
			}
			key := t.(string)
			if keys[key] {
				return fmt.Errorf("invalid json: duplicate key %s", strings.Join(append(path, key), "."))
			}
			keys[key] = true
			if err := checkDuplicateKeys(d, append(path, key)); err != nil {
				return err
			}
		}
		// Consume trailing }
		if _, err := d.Token(); err != nil {
			return err
		}
	case '[':
		i := 0
		for d.More() {
			if err := checkDuplicateKeys(d, append(path, fmt.Sprintf("[%d]", i))); err != nil {
				return err
			}
			i++
		}
		// Consume trailing ]
		if _, err := d.Token(); err != nil {
			return err
		}
	}
	return nil
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

func ValidateJSONContentType(contentTypeHeader string) error {
	isJSON, err := IsJSONContentType(contentTypeHeader)
	if err != nil {
		return err
	}
	if !isJSON {
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("unsupported media type"),
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}

func IsJSONContentType(contentTypeHeader string) (bool, error) {
	if len(contentTypeHeader) == 0 {
		return true, nil
	}

	mimeType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return false, &HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("unsupported media type"),
			StatusCode:  http.StatusBadRequest,
		}
	}

	if mimeType == jsonContentType {
		return true, nil
	}
	return false, nil
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
				Description: err.Error(),
				StatusCode:  http.StatusBadRequest,
			}
		}
	}
	return nil
}

// WriteJSON writes a JSON value and sets the specified HTTP Status code
func WriteJSON(writer http.ResponseWriter, code int, value interface{}) error {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	return encoder.Encode(value)
}

// EmptyResponseBody represents an empty response body value
type EmptyResponseBody struct{}

// NewJSONResponse turns plain object into a byte array representing JSON value and wraps it in web.Response
func NewJSONResponse(code int, value interface{}) (*web.Response, error) {
	return NewJSONResponseWithHeaders(code, value, nil)
}

func NewJSONResponseWithHeaders(code int, value interface{}, additionalHeaders map[string]string) (*web.Response, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	for header, value := range additionalHeaders {
		headers.Add(header, value)
	}

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

func NewLocationResponse(operationID, resourceID, resourceBaseURL string) (*web.Response, error) {
	operationURL := buildOperationURL(operationID, resourceID, resourceBaseURL)
	additionalHeaders := map[string]string{"Location": operationURL}
	return NewJSONResponseWithHeaders(http.StatusAccepted, map[string]string{}, additionalHeaders)
}

func buildOperationURL(operationID, resourceID, resourceType string) string {
	return fmt.Sprintf("%s/%s%s/%s", resourceType, resourceID, web.ResourceOperationsURL, operationID)
}
