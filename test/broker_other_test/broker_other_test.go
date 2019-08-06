package broker_other_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/test/common"
)

func TestBrokersOther(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Other Brokers Test Suite")
}

var _ = Describe("Other brokers", func() {
	Context("when the broker call for catalog times out", func() {
		const (
			timeoutDuration             = time.Millisecond * 500
			additionalDelayAfterTimeout = time.Second
		)

		var (
			postBrokerRequestWithNoLabels common.Object
			brokerServer                  *common.BrokerServer
			timeoutTestCtx                *common.TestContext
		)

		BeforeEach(func() {
			brokerServer = common.NewBrokerServer()
			brokerServer.Reset()

			brokerName := "brokerName"
			brokerDescription := "description"

			postBrokerRequestWithNoLabels = common.Object{
				"name":        brokerName,
				"broker_url":  brokerServer.URL(),
				"description": brokerDescription,
				"credentials": common.Object{
					"basic": common.Object{
						"username": brokerServer.Username,
						"password": brokerServer.Password,
					},
				},
			}

			timeoutTestCtx = common.NewTestContextBuilder().WithEnvPreExtensions(func(set *pflag.FlagSet) {
				Expect(set.Set("httpclient.response_header_timeout", timeoutDuration.String())).ToNot(HaveOccurred())
			}).Build()

			brokerServer.CatalogHandler = func(rw http.ResponseWriter, req *http.Request) {
				catalogStopDuration := timeoutDuration + additionalDelayAfterTimeout
				continueCtx, _ := context.WithTimeout(req.Context(), catalogStopDuration)

				<-continueCtx.Done()

				common.SetResponse(rw, http.StatusTeapot, common.Object{})
			}
		})

		AfterEach(func() {
			if brokerServer != nil {
				brokerServer.Close()
			}
			timeoutTestCtx.Cleanup()
		})

		It("returns 502", func() {
			timeoutTestCtx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(postBrokerRequestWithNoLabels).
				Expect().
				Status(http.StatusBadGateway).JSON().Object().Value("description").String().Contains("could not reach service broker")
		})
	})
})
