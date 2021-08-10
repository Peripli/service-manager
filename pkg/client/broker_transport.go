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
package client

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/Peripli/service-manager/pkg/httpclient"
	"github.com/sirupsen/logrus"
	"net/http"
)

func GetTransportWithTLS(tlsConfig *tls.Config, logger *logrus.Entry) *http.Transport {
	transport := http.Transport{}
	httpclient.ConfigureTransport(&transport)
	transport.TLSClientConfig.Certificates = tlsConfig.Certificates
	if len(tlsConfig.Certificates)>0 {
		cert, err := x509.ParseCertificate(tlsConfig.Certificates[0].Certificate[0])
		if err == nil {
			logger.Infof("sending certificate with the subject %+v,", cert.Subject.ToRDNSequence())
		}
	}

	//prevents keeping idle connections when accessing to different broker hosts
	transport.DisableKeepAlives = true
	return &transport
}
