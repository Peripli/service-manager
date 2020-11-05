package filter_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/test"
	"github.com/gofrs/uuid"
	"github.com/spf13/pflag"
	"net/http"
	"testing"
	"time"

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

type overrideFilter struct {
	ClientIP string
	UserName string
}

func (f overrideFilter) Name() string {
	return "RateLimiterOverrideFilter"
}

func (f overrideFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	if f.ClientIP != "" {
		request.Header.Set("X-Forwarded-For", f.ClientIP)
	}
	user, _ := web.UserFromContext(request.Context())
	if user != nil && f.UserName != "" {
		user.Name = f.UserName
	}
	return next.Handle(request)
}

func (f overrideFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("**/*"),
			},
		},
	}
}


var _ = Describe("Service Manager Rate Limiter", func() {
	var ctx *common.TestContext
	var osbURL string
	var serviceID string
	var planID string
	var filterContext = &overrideFilter{}
	JustBeforeEach(func() {
		ctx = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
			Expect(set.Set("api.rate_limit", "20-M")).ToNot(HaveOccurred())
		}).WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			smb.RegisterFiltersBefore("RateLimiterFilter", filterContext)
			return nil
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

		FWhen("request is authorized", func() {

			It("Authenticate with basic auth (Global Platform) - No limit to be applied", func() {
				ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).Header("X-RateLimit-Remaining").Equal("")
			})

			It("authenticate with JWT auth", func() {
				UUID, err := uuid.NewV4()
				Expect(err).ToNot(HaveOccurred())
				userName := UUID.String()
				filterContext.UserName = userName
				ctx.SMWithOAuth.GET(web.ServiceBrokersURL).
					Expect().Status(http.StatusOK).Header("X-RateLimit-Remaining").Equal("19")
				filterContext.UserName = ""
			})

			It("access a public endpoint - no limit to be applied", func() {
				ctx.SMWithBasic.GET("/v1/info").
					Expect().Status(http.StatusOK).Header("X-RateLimit-Remaining").Equal("")
			})

			It("request limit is exceeded", func() {
				UUID, err := uuid.NewV4()
				Expect(err).ToNot(HaveOccurred())
				userName := UUID.String()
				filterContext.UserName = userName
				for {
					resp := ctx.SMWithOAuth.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
					remaining := resp.Header("X-RateLimit-Remaining").Raw()
					if remaining == "0" {
						break
					}
				}
				ctx.SMWithOAuth.GET(web.ServiceBrokersURL).Expect().Status(http.StatusTooManyRequests)
				filterContext.UserName = ""
			})

			It("request limit is reset", func() {
				UUID, err := uuid.NewV4()
				Expect(err).ToNot(HaveOccurred())
				userName := UUID.String()
				filterContext.UserName = userName
				for {
					resp := ctx.SMWithOAuth.GET(web.ServiceBrokersURL).Expect().Status(http.StatusOK)
					remaining := resp.Header("X-RateLimit-Remaining").Raw()
					if remaining == "0" {
						break
					}
				}
				ctx.SMWithOAuth.GET(web.ServiceBrokersURL).Expect().Status(http.StatusTooManyRequests)
				time.Sleep(61 * time.Second)
				ctx.SMWithOAuth.GET(web.ServiceBrokersURL).
					Expect().Status(http.StatusOK).Header("X-RateLimit-Remaining").Equal("19")
				filterContext.UserName = ""
			})

		})

	})
})
