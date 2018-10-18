/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sm

import (
	"crypto/tls"
	"net/http"
)

// BasicAuthTransport implements http.RoundTripper interface and intercepts that request that is being sent,
// adding basic authorization and delegates back to the original transport.
type BasicAuthTransport struct {
	Username string
	Password string

	Rt http.RoundTripper
}

var _ http.RoundTripper = &BasicAuthTransport{}

// RoundTrip implements http.RoundTrip and adds basic authorization header before delegating to the
// underlying RoundTripper
func (b *BasicAuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if b.Username != "" && b.Password != "" {
		request.SetBasicAuth(b.Username, b.Password)
	}

	return b.Rt.RoundTrip(request)
}

// SkipSSLTransport implements http.RoundTripper and sets the SSL Validation to match the provided property
type SkipSSLTransport struct {
	SkipSslValidation bool

	Rt http.RoundTripper
}

var _ http.RoundTripper = &SkipSSLTransport{}

// RoundTrip implements http.RoundTrip and adds skip SSL validation logic
func (b *SkipSSLTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if b.Rt == nil {
		b.Rt = http.DefaultTransport
	}

	if defaultTransport, ok := b.Rt.(*http.Transport); ok {
		defaultTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: b.SkipSslValidation,
		}
	}

	return b.Rt.RoundTrip(request)
}
