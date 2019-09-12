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

package httpclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Settings struct {
	TLSHandshakeTimeout   time.Duration `mapstructure:"tls_handshake_timeout"`
	IdleConnTimeout       time.Duration `mapstructure:"idle_conn_timeout"`
	ResponseHeaderTimeout time.Duration `mapstructure:"response_header_timeout"`
	DialTimeout           time.Duration `mapstructure:"dial_timeout"`
	SkipSSLValidation     bool          `mapstructure:"skip_ssl_validation" description:"whether to skip ssl verification when making calls to external services"`
}

// DefaultSettings return the default values for httpclient settings
func DefaultSettings() *Settings {
	return &Settings{
		TLSHandshakeTimeout:   time.Second * 10,
		IdleConnTimeout:       time.Second * 10,
		ResponseHeaderTimeout: time.Second * 10,
		DialTimeout:           time.Second * 10,
		SkipSSLValidation:     false,
	}
}

// Validate validates the httpclient settings
func (s *Settings) Validate() error {
	if s.TLSHandshakeTimeout < 0 {
		return fmt.Errorf("validate httpclient settings: tls_handshake_timeout should be >= 0")
	}
	if s.IdleConnTimeout < 0 {
		return fmt.Errorf("validate httpclient settings: idle_conn_timeout should be >= 0")
	}
	if s.ResponseHeaderTimeout < 0 {
		return fmt.Errorf("validate httpclient settings: response_header_timeout should be >= 0")
	}
	if s.DialTimeout < 0 {
		return fmt.Errorf("validate httpclient settings: dial_timeout should be >= 0")
	}
	return nil
}

// Configure configures the default http client and transport
func Configure(settings *Settings) {
	transport := http.DefaultTransport.(*http.Transport)

	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: settings.SkipSSLValidation}
	transport.ResponseHeaderTimeout = settings.ResponseHeaderTimeout
	transport.TLSHandshakeTimeout = settings.TLSHandshakeTimeout
	transport.IdleConnTimeout = settings.IdleConnTimeout
	transport.DialContext = (&net.Dialer{Timeout: settings.DialTimeout}).DialContext

	http.DefaultClient.Transport = transport
}
