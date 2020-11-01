package filter_test

import (
	"github.com/Peripli/service-manager/test"
	"github.com/gofrs/uuid"
	"github.com/spf13/pflag"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type object = common.Object

func TestFilters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rate Limiter Tests Suite")
}

var _ = Describe("Service Manager Rate Limiter", func() {
	var ctx *common.TestContext
	var osbURL string
	var serviceID string
	var planID string

	JustBeforeEach(func() {
		ctx = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
			Expect(set.Set("api.rate_limit", "20-M")).ToNot(HaveOccurred())
		}).Build()

		UUID, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		planID = UUID.String()
		plan1 := common.GenerateTestPlanWithID(planID)
		UUID, err = uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		serviceID = UUID.String()
		service1 := common.GenerateTestServiceWithPlansWithID(serviceID, plan1)
		catalog := common.NewEmptySBCatalog()
		catalog.AddService(service1)
		brokerID := ctx.RegisterBrokerWithCatalog(catalog).Broker.ID
		common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, brokerID)

		username, password := test.RegisterBrokerPlatformCredentials(ctx.SMWithBasic, brokerID)
		ctx.SMWithBasic.SetBasicCredentials(ctx, username, password)
		osbURL = "/v1/osb/" + brokerID
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	Describe("rate limiter", func() {

		FWhen("request is authorized ", func() {

			It("authenticate with a JWT token", func() {
				ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).Header("X-RateLimit-Remaining").Equal("19")
			})

			It("authenticate with basic auth", func() {
				ctx.SMWithOAuth.GET(web.ServiceBrokersURL).
					Expect().Status(http.StatusOK).Header("X-RateLimit-Remaining").Equal("13")
			})

			It("request limit is exceeded", func() {

			})

			It("request limit is reset", func() {

			})

		})

		When("request anonymous request is rejected", func() {
			It("request limit is not exceeded", func() {

			})

			It("request limit is exceeded", func() {

			})

			It("request limit is reset", func() {

			})

		})
	})
})
