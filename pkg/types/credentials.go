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

package types

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/Peripli/service-manager/pkg/httpclient"
)

// Basic basic credentials
type Basic struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type TLS struct {
	Certificate      string `json:"client_certificate,omitempty"`
	Key              string `json:"client_key,omitempty"`
	UseSMCertificate bool   `json:"sm_provided_tls_credentials,omitempty"`
}

// Credentials credentials
type Credentials struct {
	Basic     *Basic `json:"basic,omitempty"`
	TLS       *TLS   `json:"tls,omitempty"`
	Integrity []byte `json:"-"`
}

func (c *Credentials) MarshalJSON() ([]byte, error) {
	type C Credentials
	toMarshal := (*C)(c)
	if toMarshal.Basic == nil || toMarshal.Basic.Username == "" || toMarshal.Basic.Password == "" {
		toMarshal.Basic = nil
	}

	return json.Marshal(toMarshal)
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (c *Credentials) Validate() error {
	if !c.BasicExists() && (*c.TLS == TLS{}) {
		return errors.New("missing broker credentials: SM provided mtls , basic or tls credentials are required")
	}
	if c.BasicExists() {
		err := c.validateBasic()
		if err != nil {
			return err
		}
	}
	if c.CertificateExists() {
		err := c.validateTLS()
		if err != nil {
			return err
		}
	}
	if c.TLS != nil && c.TLS.UseSMCertificate {
		err := c.validateSMProvidedCredentials()
		if err != nil {
			return err
		}
	}

	return nil
}
func (c *Credentials) validateSMProvidedCredentials() error {
	httpSettings := httpclient.GetHttpClientGlobalSettings()
	isMTLSEnabled := len(httpSettings.ServerCertificate) > 0
	if !isMTLSEnabled {
		return errors.New("SM provided credentials are not supported in this region, set another type of credentials")
	}
	return nil
}

func (c *Credentials) validateTLS() error {
	if c.TLS.Certificate == "" || c.TLS.Key == "" {
		return errors.New("tls public certificate and key should be provided")
	}

	if c.TLS.UseSMCertificate {
		return errors.New("only one of the options could be set, SM provided credentials or tls")
	}
	_, err := tls.X509KeyPair([]byte(c.TLS.Certificate), []byte(c.TLS.Key))
	if err != nil {
		return errors.New("invalidate TLS configuration: " + err.Error())
	}

	return nil
}

func (c *Credentials) validateBasic() error {
	if c.Basic.Username == "" {
		return errors.New("missing broker username")
	}
	if c.Basic.Password == "" {
		return errors.New("missing broker password")
	}

	return nil
}

// GenerateCredentials return user and password
func GenerateCredentials() (*Credentials, error) {
	password := make([]byte, 32)
	user := make([]byte, 32)

	_, err := rand.Read(user)
	if err != nil {
		return nil, err
	}
	_, err = rand.Read(password)
	if err != nil {
		return nil, err
	}

	encodedPass := base64.StdEncoding.EncodeToString(password)
	encodedUser := base64.StdEncoding.EncodeToString(user)

	return &Credentials{
		Basic: &Basic{
			Username: encodedUser,
			Password: encodedPass,
		},
	}, nil
}

func (c *Credentials) CertificateExists() bool {
	return c.TLS != nil && !(c.TLS.Certificate == "" && c.TLS.Key == "")
}

func (c *Credentials) BasicExists() bool {
	return c.Basic != nil && *c.Basic != Basic{}
}
