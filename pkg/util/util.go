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
	"crypto/tls"
	"net/http"
)

// DefaultHTTPClient constructs a default HTTP Client with a specific certificate validation configuration
func DefaultHTTPClient(skipSSLValidation bool) *http.Client {
	client := http.DefaultClient
	client.Transport = defaultHTTPTransport(skipSSLValidation)
	return client
}

func defaultHTTPTransport(skipSSLValidation bool) http.RoundTripper {
	defaultTransport := http.DefaultTransport.(*http.Transport)

	transportWithSelfSignedTLS := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		DialContext:           defaultTransport.DialContext,
		MaxIdleConns:          defaultTransport.MaxIdleConns,
		IdleConnTimeout:       defaultTransport.IdleConnTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: skipSSLValidation},
	}

	return transportWithSelfSignedTLS
}
