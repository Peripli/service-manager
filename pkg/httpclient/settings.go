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
	"crypto/x509"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"net"
	"net/http"
	"time"
)

type Settings struct {
	Timeout               time.Duration `mapstructure:"timeout" description:"timeout specifies a time limit for the request. The timeout includes connection time, any redirects, and reading the response body"`
	TLSHandshakeTimeout   time.Duration `mapstructure:"tls_handshake_timeout"`
	ServerCertificateKey  string        `mapstructure:"server_certificate_key"`
	ServerCertificate     string        `mapstructure:"server_certificate"`
	IdleConnTimeout       time.Duration `mapstructure:"idle_conn_timeout"`
	ResponseHeaderTimeout time.Duration `mapstructure:"response_header_timeout"`
	DialTimeout           time.Duration `mapstructure:"dial_timeout"`
	SkipSSLValidation     bool          `mapstructure:"skip_ssl_validation" description:"whether to skip ssl verification when making calls to external services"`
	RootCACertificates    []string      `mapstructure:"root_certificates"`

	TLSCertificates []tls.Certificate
}

var globalSettings Settings

// DefaultSettings return the default values for httpclient settings
func DefaultSettings() *Settings {
	return &Settings{
		Timeout:               time.Second * 15,
		TLSHandshakeTimeout:   time.Second * 10,
		IdleConnTimeout:       time.Second * 10,
		ResponseHeaderTimeout: time.Second * 10,
		DialTimeout:           time.Second * 10,
		SkipSSLValidation:     false,
	}
}

// Validate validates the httpclient settings
func (s *Settings) Validate() error {
	if s.Timeout < 0 {
		return fmt.Errorf("validate httpclient settings: timeout should be >= 0")
	}
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
	if s.ServerCertificate != "" && s.ServerCertificateKey != "" {
		_, err := tls.X509KeyPair([]byte(s.ServerCertificate), []byte(s.ServerCertificateKey))
		if err != nil {
			return fmt.Errorf("bad certificate: %s", err)
		}
		log.D().Info("client certificate is ok ")
	}

	return nil
}

func GetHttpClientGlobalSettings() *Settings {
	return &globalSettings
}

func SetHTTPClientGlobalSettings(settings *Settings) {
	globalSettings = *settings
}

// Configures the http client transport
func Configure() {
	settings := GetHttpClientGlobalSettings()
	http.DefaultClient.Timeout = settings.Timeout
	ConfigureTransport(http.DefaultTransport.(*http.Transport))
}

func ConfigureTransport(transport *http.Transport) {
	settings := GetHttpClientGlobalSettings()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: settings.SkipSSLValidation}
	if settings.ServerCertificate!="" && settings.ServerCertificateKey!=""{
		cert, _:= tls.X509KeyPair([]byte(settings.ServerCertificate), []byte(settings.ServerCertificateKey))
		settings.TLSCertificates = []tls.Certificate{cert}
		if len(settings.TLSCertificates) > 0 {
			transport.TLSClientConfig.Certificates = settings.TLSCertificates
		}
	}


	if len(settings.RootCACertificates) > 0 {
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			caCertPool = x509.NewCertPool()
		}
		for _, certificate := range settings.RootCACertificates {
			caCertPool.AppendCertsFromPEM([]byte(certificate))
		}
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	transport.ResponseHeaderTimeout = settings.ResponseHeaderTimeout
	transport.TLSHandshakeTimeout = settings.TLSHandshakeTimeout
	transport.IdleConnTimeout = settings.IdleConnTimeout
	transport.DialContext = (&net.Dialer{Timeout: settings.DialTimeout}).DialContext
}
