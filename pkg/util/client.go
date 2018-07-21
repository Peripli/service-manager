package util

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"io/ioutil"
)

// SendClientRequest sends a request to the specified client and the provided URL with the specified parameters and body.
func SendClientRequest(client *http.Client, method, URL string, params map[string]string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	request, err := http.NewRequest(method, URL, bodyReader)

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

	return client.Do(request)
}

// ReadClientResponseContent of the request inside given struct
func ReadClientResponseContent(value interface{}, closer io.ReadCloser) error {
	body, err := ioutil.ReadAll(closer)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, value)
	return err
}
