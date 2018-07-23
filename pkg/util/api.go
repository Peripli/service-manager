package util

import (
	"encoding/json"
	"net/http"
	"time"

	"io/ioutil"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

var (
	reservedSymbolsRFC3986 = strings.Join([]string{
		":", "/", "?", "#", "[", "]", "@", "!", "$", "&", "'", "(", ")", "*", "+", ",", ";", "=",
	}, "")
)

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

func ReadHTTPRequestBody(request *http.Request) ([]byte, error) {
	contentType := request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, &HTTPError{
			ErrorType:   "InvalidMediaType",
			Description: "invalid media type provided",
			StatusCode:  http.StatusUnsupportedMediaType,
		}
	}
	body, err := ioutil.ReadAll(request.Body)
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

// UnmarshalAndValidate parses and validates the request body
func UnmarshalAndValidate(body []byte, value interface{}) error {
	if err := Unmarshal(body, value); err != nil {
		return err
	}
	if err := Validate(value); err != nil {
		return err
	}

	return nil
}

// Unmarshal unmarshals the specified []byte into the provided value and returns an HttpError in unmarshaling fails
func Unmarshal(body []byte, value interface{}) error {
	err := json.Unmarshal(body, value)
	if err != nil {
		logrus.Error("Failed to decode request body: ", err)
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: "Failed to decode request body",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}

// Validate validates the specified value in case it implements InputValidator
func Validate(value interface{}) error {
	if input, ok := value.(InputValidator); ok {
		return &HTTPError{
			ErrorType:   "BadRequest",
			Description: input.Validate().Error(),
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}

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
