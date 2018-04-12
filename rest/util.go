package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// SendJSON writes a JSON value and sets the specified HTTP Status code
func SendJSON(writer http.ResponseWriter, code int, value interface{}) error {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(code)

	encoder := json.NewEncoder(writer)
	return encoder.Encode(value)
}

func ReadJSONBody(request *http.Request, value interface{}) error {
	contentType := request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return CreateErrorResponse(errors.New("Invalid media type provided"), http.StatusUnsupportedMediaType, "InvalidMediaType")
	}
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	if err := decoder.Decode(value); err != nil {
		return CreateErrorResponse(errors.New("Invalid JSON"), http.StatusBadRequest, "InvalidJSON")
	}
	return nil
}
