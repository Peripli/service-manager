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

package util

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"io/ioutil"

	"github.com/sirupsen/logrus"
)

// DoRequestFunc is an alias for any function that takes an http request and returns a response and error
type DoRequestFunc func(request *http.Request) (*http.Response, error)

// SendRequest sends a request to the specified client and the provided URL with the specified parameters and body.
func SendRequest(doRequest DoRequestFunc, method, url string, params map[string]string, body interface{}) (*http.Response, error) {
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

// BodyToBytes of the request inside given struct
func BodyToBytes(closer io.ReadCloser) ([]byte, error) {
	defer func() {
		if err := closer.Close(); err != nil {
			logrus.Errorf("ReadCloser couldn't be closed", err)
		}
	}()

	body, err := ioutil.ReadAll(closer)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// BodyToObject of the request inside given struct
func BodyToObject(closer io.ReadCloser, object interface{}) error {
	body, err := BodyToBytes(closer)
	if err != nil {
		return err
	}
	return BytesToObject(body, object)
}
