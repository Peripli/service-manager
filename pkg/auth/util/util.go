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
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/auth"
	"net"
	"net/http"
	"net/url"
	"time"
)

// ValidateURL validates a URL
func ValidateURL(URL string) error {
	if URL == "" {
		return errors.New("url not provided")
	}

	parsedURL, err := url.Parse(URL)
	if err != nil {
		return fmt.Errorf("url cannot be parsed: %s", err)
	}

	if !parsedURL.IsAbs() || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("url is not an HTTP URL: %s", URL)
	}

	return nil
}

// BuildHTTPClient builds custom http client with configured ssl validation / mtls
func BuildHTTPClient(options *auth.Options) (*http.Client, error) {
	client := getClient()

	if MtlsEnabled(options) {
		cert, err := tls.LoadX509KeyPair(options.Certificate, options.Key)
		if err != nil {
			return nil, err
		}

		client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		return client, nil
	} else {
		if options.SSLDisabled {
			client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}

	return client, nil
}

func MtlsEnabled(options *auth.Options) bool {
	return len(options.Certificate) > 0 && len(options.Key) > 0
}

func getClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}
