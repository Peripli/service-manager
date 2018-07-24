package util

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"io/ioutil"
)

type doRequestFunc func(request *http.Request) (*http.Response, error)

// SendClientRequest sends a request to the specified client and the provided URL with the specified parameters and body.
func SendClientRequest(doRequest doRequestFunc, method, url string, params map[string]string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	request, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if params != nil {
		q := request.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		request.URL.RawQuery = q.Encode()
	}

	return doRequest(request)
}

// ReadClientResponseContent of the request inside given struct
func ReadClientResponseContent(value interface{}, closer io.ReadCloser) error {
	body, err := ioutil.ReadAll(closer)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, value)
}
