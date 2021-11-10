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
	"crypto/tls"
	"fmt"
	"github.com/Peripli/service-manager/pkg/auth"
	"net"
	"net/http"
	"strings"
	"time"
)

// BuildHTTPClient builds custom http client with configured ssl validation / mtls
func BuildHTTPClient(options *auth.Options) (*http.Client, error) {
	if options.MtlsEnabled() {
		return newMtlsClient(options)
	} else {
		return newClient(options)
	}
}

func newClient(options *auth.Options) (*http.Client, error) {
	client := getClient()
	if options.SSLDisabled {
		client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return client, nil
}

func newMtlsClient(options *auth.Options) (*http.Client, error) {
	var err error
	var cert tls.Certificate

	client := getClient()

	certBytes := []byte(options.Certificate)
	keyBytes := []byte(options.Key)

	if pemFile(certBytes) != pemFile(keyBytes) {
		return nil, fmt.Errorf("both certificate and key must be provided as file or as string")
	}
	if pemFile(certBytes) && pemFile(keyBytes) {
		cert, err = tls.LoadX509KeyPair(options.Certificate, options.Key)
	} else {
		cert, err = tls.X509KeyPair(certBytes, keyBytes)
	}
	if err != nil {
		return nil, err
	}

	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return client, nil
}

func pemFile(data []byte) bool {
	fileType := []byte(".pem")
	return bytes.HasSuffix(data, fileType)
}

func ConvertBackSlashN(originalValue string) string {
	lines := strings.Split(originalValue, `\n`)
	var value string
	for _, line := range lines {
		value += fmt.Sprintln(line)
	}
	return value
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
