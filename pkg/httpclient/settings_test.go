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
	"github.com/Peripli/service-manager/test/tls_settings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTPClient settings suite")
}

var _ = Describe("HTTPClient settings", func() {
	var settings *Settings

	BeforeEach(func() {
		settings = DefaultSettings()
	})

	assertValidateError := func(errMessage string) {
		err := settings.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errMessage))
	}

	Describe("Validate", func() {
		Context("on valid settings", func() {
			It("should return nil", func() {
				Expect(settings.Validate()).ToNot(HaveOccurred())
			})
		})
		Context("valid server certificate", func() {
			It("should have certificate", func() {
				settings.ServerCertificate = tls_settings.ServerManagerCertificate
				settings.ServerCertificateKey = tls_settings.ServerManagerCertificateKey
				Expect(settings.Validate()).ToNot(HaveOccurred())

			})
		})

		Context("invalid server certificate", func() {
			It("should have certificate", func() {
				settings.ServerCertificate = tls_settings.InvalidServerManagerCertificate
				settings.ServerCertificateKey = tls_settings.InvalidServerManagerCertificateKey
				Expect(settings.Validate()).To(HaveOccurred())
			})
		})

		Context("on invalid tls_handshake_timeout", func() {
			It("should return error", func() {
				settings.TLSHandshakeTimeout = -1
				assertValidateError("validate httpclient settings: tls_handshake_timeout should be >= 0")
			})
		})

		Context("on invalid dial_timeout", func() {
			It("should return error", func() {
				settings.DialTimeout = -1
				assertValidateError("validate httpclient settings: dial_timeout should be >= 0")
			})
		})

		Context("on invalid response_header_timeout", func() {
			It("should return error", func() {
				settings.ResponseHeaderTimeout = -1
				assertValidateError("validate httpclient settings: response_header_timeout should be >= 0")
			})
		})

		Context("on invalid idle_conn_timeout", func() {
			It("should return error", func() {
				settings.IdleConnTimeout = -1
				assertValidateError("validate httpclient settings: idle_conn_timeout should be >= 0")
			})
		})
	})
})
