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
	var changeContextUser = func() {
		UUID, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		userName := UUID.String()
		filterContext.UserName = userName
	}
	var newRateLimiterEnv = func(limit string, excludePaths string) {
		ctx = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
			Expect(set.Set("api.rate_limit", limit)).ToNot(HaveOccurred())
			Expect(set.Set("api.rate_limiting_enabled", "true")).ToNot(HaveOccurred())
			Expect(set.Set("api.rate_limit_exclude_paths", excludePaths)).ToNot(HaveOccurred())
		}).WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			smb.RegisterFiltersBefore("RateLimiterFilter", filterContext)
			return nil
		}).Build()
	}
	var expectLimitedRequest = func(expect *common.SMExpect, path string) {
		expect.GET(path).Expect().Status(http.StatusTooManyRequests)
	}
	var expectNonLimitedRequest = func(expect *common.SMExpect, path string) {
		expect.GET(path).Expect().Status(http.StatusOK)
	}
	var bulkRequest = func(expect *common.SMExpect, path string, times int) {
		for i := 1; i <= times; i++ {
			expectNonLimitedRequest(expect, path)
		}
	}

	JustBeforeEach(func() {
		newRateLimiterEnv("20-M,20-M", "")
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
		filterContext.UserName = ""
	})

	Describe("rate limiter", func() {

		FWhen("request is authorized", func() {

			It("Authenticate with basic auth (Global Platform) - No limit to be applied", func() {
				bulkRequest(ctx.SMWithBasic, osbURL+"/v2/catalog", 100)
			})

			It("access a public endpoint - no limit to be applied", func() {
				bulkRequest(ctx.SMWithBasic, "/v1/info", 100)
				bulkRequest(ctx.SMWithOAuth, "/v1/info", 100)
			})

			It("access a excluded endpoint - no limit to be applied", func() {
				newRateLimiterEnv("20-M,20-M", web.PlatformsURL)
				changeContextUser()
				bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 100)
				// Call for non excluded path should be limited
				bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
				expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
			})

			It("request limit is exceeded", func() {
				changeContextUser()
				bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
				expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
			})

			It("request limit is reset", func() {
				changeContextUser()
				bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
				expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				time.Sleep(61 * time.Second)
				bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 3)
			})

			It("secondary limiter test", func() {
				newRateLimiterEnv("20-S,30-M", "")
				changeContextUser()
				// Exhaust seconds limiter, minute limiter drop from 30 to 10
				bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
				time.Sleep(1 * time.Second)
				// Exhaust minute limiter
				bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 10)
				// Expecting second limiter will reset, but minute should be still exhausted
				time.Sleep(3 * time.Second)
				expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				// Expecting all limiters will reset
				time.Sleep(61 * time.Second)
				expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
			})

		})

	})
})
