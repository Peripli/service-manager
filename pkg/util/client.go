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
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
)

// DoRequestFunc is an alias for any function that takes an http request and returns a response and error
type DoRequestFunc func(request *http.Request) (*http.Response, error)

func BasicAuthDecorator(username, password string, requestFunc DoRequestFunc) DoRequestFunc {
	return func(req *http.Request) (*http.Response, error) {
		req.SetBasicAuth(username, password)
		return requestFunc(req)
	}
}

// SendRequest sends a request to the specified client and the provided URL with the specified parameters and body.
func SendRequest(ctx context.Context, doRequest DoRequestFunc, method, url string, params map[string]string, body interface{}) (*http.Response, error) {
	return SendRequestWithHeaders(ctx, doRequest, method, url, params, body, map[string]string{})
}

// SendRequestWithHeaders sends a request to the specified client and the provided URL with the specified parameters, body and headers.
func SendRequestWithHeaders(ctx context.Context, doRequest DoRequestFunc, method, url string, params map[string]string, body interface{}, headers map[string]string) (*http.Response, error) {
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

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	if params != nil {
		q := request.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		request.URL.RawQuery = q.Encode()
	}

	request = request.WithContext(ctx)
	logger := log.C(ctx)
	correlationID, exists := logger.Data[log.FieldCorrelationID].(string)
	if exists {
		request.Header.Set(log.CorrelationIDHeaders[0], correlationID)
	}

	logger.Debugf("Sending a request to %s", request.URL)
	return doRequest(request)
}

// BodyToBytes of the request inside given struct
func BodyToBytes(closer io.ReadCloser) ([]byte, error) {
	defer func() {
		if err := closer.Close(); err != nil {
			log.D().Errorf("ReadCloser couldn't be closed: %v", err)
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
