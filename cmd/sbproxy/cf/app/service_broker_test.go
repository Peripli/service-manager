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

var _ = Describe("Client ServiceBroker", func() {
	var (
		config          *ClientConfiguration
		client          *PlatformClient
		ccServer        *ghttp.Server
		testBroker      *platform.ServiceBroker
		ccResponseCode  int
		ccResponse      interface{}
		ccResponseErr   CloudFoundryErr
		expectedRequest interface{}
		ctx             context.Context
	)

	assertBrokersFoundMatchTestBroker := func(expectedCount int, actualBrokers ...platform.ServiceBroker) {
		Expect(actualBrokers).To(HaveLen(expectedCount))
		for _, b := range actualBrokers {
			Expect(&b).To(Equal(testBroker))
		}
	}

	BeforeEach(func() {
		ctx = context.TODO()

		testBroker = &platform.ServiceBroker{
			GUID:      "test-testBroker-guid",
			Name:      "test-testBroker-name",
			BrokerURL: "http://example.com",
		}

		ccServer = fakeCCServer(false)

		config, client = ccClient(ccServer.URL())

	})

	AfterEach(func() {
		ccServer.Close()
	})

	Describe("GetBrokers", func() {
		BeforeEach(func() {
			ccServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v2/service_brokers"),
					ghttp.RespondWithJSONEncodedPtr(&ccResponseCode, &ccResponse),
				),
			)
		})

		Context("when an error status code is returned by CC", func() {
			BeforeEach(func() {
				ccResponseErr = CloudFoundryErr{
					Code:        1009,
					ErrorCode:   "err",
					Description: "test err",
				}
				ccResponse = ccResponseErr

				ccResponseCode = http.StatusInternalServerError
			})

			It("returns an error", func() {
				_, err := client.GetBrokers(ctx)

				assertErrIsCFError(err, ccResponseErr)
			})

		})

		Context("when no brokers are found in CC", func() {
			BeforeEach(func() {
				ccResponse = cfclient.ServiceBrokerResponse{
					Count:     0,
					Pages:     1,
					Resources: []cfclient.ServiceBrokerResource{},
				}
				ccResponseCode = http.StatusOK
			})

			It("returns an empty slice", func() {
				brokers, err := client.GetBrokers(ctx)

				Expect(err).ShouldNot(HaveOccurred())
				assertBrokersFoundMatchTestBroker(0, brokers...)
			})

		})

		Context("when brokers exist in CC", func() {
			BeforeEach(func() {
				ccResponse = cfclient.ServiceBrokerResponse{
					Count: 1,
					Pages: 1,
					Resources: []cfclient.ServiceBrokerResource{
						{
							Meta: cfclient.Meta{
								Guid: testBroker.GUID,
							},
							Entity: cfclient.ServiceBroker{
								Name:      testBroker.Name,
								BrokerURL: testBroker.BrokerURL,
								Username:  config.Reg.User,
							},
						},
					},
				}
				ccResponseCode = http.StatusOK
			})

			It("returns all of the brokers", func() {
				brokers, err := client.GetBrokers(ctx)

				Expect(err).ShouldNot(HaveOccurred())
				assertBrokersFoundMatchTestBroker(1, brokers...)
			})
		})
	})

	Describe("CreateBroker", func() {
		var actualRequest *platform.CreateServiceBrokerRequest

		BeforeEach(func() {
			expectedRequest = &cfclient.CreateServiceBrokerRequest{
				Name:      testBroker.Name,
				BrokerURL: testBroker.BrokerURL,
				Username:  config.Reg.User,
				Password:  config.Reg.Password,
			}

			actualRequest = &platform.CreateServiceBrokerRequest{
				Name:      testBroker.Name,
				BrokerURL: testBroker.BrokerURL,
			}

			ccServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v2/service_brokers"),
					ghttp.VerifyJSONRepresenting(expectedRequest),
					ghttp.RespondWithJSONEncodedPtr(&ccResponseCode, &ccResponse),
				),
			)
		})

		Context("when an error status code is returned by CC", func() {
			BeforeEach(func() {
				ccResponseErr = CloudFoundryErr{
					Code:        1009,
					ErrorCode:   "err",
					Description: "test err",
				}

				ccResponse = ccResponseErr
				ccResponseCode = http.StatusInternalServerError
			})

			It("returns an error", func() {
				_, err := client.CreateBroker(ctx, actualRequest)

				assertErrIsCFError(err, ccResponseErr)
			})
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				ccResponseCode = http.StatusCreated
				ccResponse = cfclient.ServiceBrokerResource{
					Meta: cfclient.Meta{
						Guid: testBroker.GUID,
					},
					Entity: cfclient.ServiceBroker{
						Name:      testBroker.Name,
						BrokerURL: testBroker.BrokerURL,
						Username:  config.Reg.User,
					},
				}
			})

			It("returns the created broker", func() {
				broker, err := client.CreateBroker(ctx, actualRequest)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(broker).To(Equal(testBroker))
			})
		})
	})

	Describe("DeleteBroker", func() {
		var actualRequest *platform.DeleteServiceBrokerRequest

		BeforeEach(func() {
			actualRequest = &platform.DeleteServiceBrokerRequest{
				GUID: testBroker.GUID,
				Name: testBroker.Name,
			}

			ccServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/v2/service_brokers/"+testBroker.GUID),
					ghttp.RespondWithJSONEncodedPtr(&ccResponseCode, &ccResponse),
				),
			)
		})

		Context("when an error status code is returned by CC", func() {
			BeforeEach(func() {
				ccResponseErr = CloudFoundryErr{
					Code:        1009,
					ErrorCode:   "err",
					Description: "test err",
				}

				ccResponse = ccResponseErr
				ccResponseCode = http.StatusInternalServerError
			})

			It("returns an error", func() {
				err := client.DeleteBroker(ctx, actualRequest)

				assertErrIsCFError(err, ccResponseErr)
			})
		})

		Context("when the broker exists in CC", func() {
			BeforeEach(func() {
				ccResponseCode = http.StatusNoContent
				ccResponse = nil
			})

			It("returns no error", func() {
				err := client.DeleteBroker(ctx, actualRequest)

				Expect(err).ShouldNot(HaveOccurred())
			})

		})

	})

	Describe("UpdateBroker", func() {
		var actualRequest *platform.UpdateServiceBrokerRequest

		BeforeEach(func() {
			expectedRequest = &cfclient.UpdateServiceBrokerRequest{
				Name:      testBroker.Name,
				BrokerURL: testBroker.BrokerURL,
				Username:  config.Reg.User,
				Password:  config.Reg.Password,
			}

			actualRequest = &platform.UpdateServiceBrokerRequest{
				GUID:      testBroker.GUID,
				Name:      testBroker.Name,
				BrokerURL: testBroker.BrokerURL,
			}

			ccServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/v2/service_brokers/"+testBroker.GUID),
					ghttp.VerifyJSONRepresenting(expectedRequest),
					ghttp.RespondWithJSONEncodedPtr(&ccResponseCode, &ccResponse),
				),
			)
		})
		Context("when an error status code is returned by CC", func() {
			BeforeEach(func() {
				ccResponseErr = CloudFoundryErr{
					Code:        1009,
					ErrorCode:   "err",
					Description: "test err",
				}

				ccResponse = ccResponseErr
				ccResponseCode = http.StatusInternalServerError
			})

			It("returns an error", func() {
				_, err := client.UpdateBroker(ctx, actualRequest)

				assertErrIsCFError(err, ccResponseErr)
			})
		})

		Context("when the request is successful", func() {
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

			It("returns the updated broker", func() {
				broker, err := client.UpdateBroker(ctx, actualRequest)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(broker).Should(Equal(testBroker))
			})
		})

	})
})
