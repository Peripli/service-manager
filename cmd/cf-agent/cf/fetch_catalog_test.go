package cf_test

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/cmd/cf-agent/cf"
	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client FetchCatalog", func() {
	var (
		settings        *cf.Settings
		client          *cf.PlatformClient
		ccServer        *ghttp.Server
		testBroker      *platform.ServiceBroker
		ccResponseCode  int
		ccResponse      interface{}
		ccResponseErr   cf.CloudFoundryErr
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

		settings, client = ccClient(ccServer.URL())

		expectedRequest = &cfclient.UpdateServiceBrokerRequest{
			Name:      testBroker.Name,
			BrokerURL: testBroker.BrokerURL,
			Username:  settings.Sm.User,
			Password:  settings.Sm.Password,
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

	It("is not nil", func() {
		Expect(client.CatalogFetcher()).ToNot(BeNil())
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
				ccResponseErr = cf.CloudFoundryErr{
					Code:        1009,
					ErrorCode:   "err",
					Description: "test err",
				}
				ccResponse = ccResponseErr

				ccResponseCode = http.StatusInternalServerError
			})

			It("propagates the error", func() {
				err = client.Fetch(ctx, testBroker)

				assertErrCauseIsCFError(err, ccResponseErr)
			})
		})
	})
})
