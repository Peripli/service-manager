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

package app

import (
	"context"
	"net/http"

	"github.com/cloudfoundry-community/go-cfclient"

	"github.com/Peripli/service-manager/pkg/sbproxy/platform"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client FetchCatalog", func() {
	var (
		config          *ClientConfiguration
		client          *PlatformClient
		ccServer        *ghttp.Server
		testBroker      *platform.ServiceBroker
		ccResponseCode  int
		ccResponse      interface{}
		ccResponseErr   CloudFoundryErr
		expectedRequest interface{}
		err             error
		ctx             context.Context
	)

	BeforeEach(func() {
		ctx = context.TODO()

		testBroker = &platform.ServiceBroker{
			GUID:      "test-testBroker-guid",
			Name:      "test-testBroker-name",
			BrokerURL: "http://example.com",
		}

		ccServer = fakeCCServer(false)

		config, client = ccClient(ccServer.URL())

		expectedRequest = &cfclient.UpdateServiceBrokerRequest{
			Name:      testBroker.Name,
			BrokerURL: testBroker.BrokerURL,
			Username:  config.Reg.User,
			Password:  config.Reg.Password,
		}

		ccServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("PUT", "/v2/service_brokers/"+testBroker.GUID),
				ghttp.VerifyJSONRepresenting(expectedRequest),
				ghttp.RespondWithJSONEncodedPtr(&ccResponseCode, &ccResponse),
			),
		)
	})

	AfterEach(func() {
		ccServer.Close()
	})

	Describe("Fetch", func() {
		Context("when the call to UpdateBroker is successful", func() {
			BeforeEach(func() {
				ccResponse = cfclient.ServiceBrokerResource{
					Meta: cfclient.Meta{
						Guid: testBroker.GUID,
					},
					Entity: cfclient.ServiceBroker{
						Name:      testBroker.Name,
						BrokerURL: testBroker.BrokerURL,
						Username:  testBroker.Name,
					},
				}

				ccResponseCode = http.StatusOK
			})

			It("returns no error", func() {
				err = client.Fetch(ctx, testBroker)

				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when UpdateBroker returns an error", func() {
			BeforeEach(func() {
				ccResponseErr = CloudFoundryErr{
					Code:        1009,
					ErrorCode:   "err",
					Description: "test err",
				}
				ccResponse = ccResponseErr

				ccResponseCode = http.StatusInternalServerError
			})

			It("propagates the error", func() {
				err = client.Fetch(ctx, testBroker)

				assertErrIsCFError(err, ccResponseErr)
			})
		})
	})

})
