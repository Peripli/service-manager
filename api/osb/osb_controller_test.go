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
package osb

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/tls_settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"log"
	"net/url"
)

var _ = Describe("OSB Controller test", func() {

	var brokerTLS types.ServiceBroker

	BeforeEach(func() {
		brokerTLS = types.ServiceBroker{
			Base: types.Base{
				ID:     "123",
				Labels: map[string][]string{},
				Ready:  true,
			},
			Name:      "tls-broker",
			BrokerURL: "url",
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: "user",
					Password: "pass",
				},
				TLS: &types.TLS{
					Certificate: tls_settings.ClientCertificate,
					Key:         tls_settings.ClientKey,
				},
			},
		}
	})

	Describe("test osb create proxy", func() {
		logger := logrus.Entry{}
		targetBrokerURL, err := url.Parse("http://example.com/proxy/")
		if err != nil {
			log.Fatal(err)
		}

		It("create proxy with tls should return a new reverse proxy with its own tls setting", func() {
			reverseProxy, _ := buildProxy(targetBrokerURL, &logger, &brokerTLS)
			Expect(reverseProxy).NotTo(Equal(nil))
			reverseProxy2, _ := buildProxy(targetBrokerURL, &logger, &brokerTLS)
			Expect(reverseProxy2.Transport).ToNot(BeIdenticalTo(reverseProxy.Transport))
		})
	})
})
