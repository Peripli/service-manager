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
	var changeClientIdentifier = func() string {
		UUID, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		userName := UUID.String()
		filterContext.UserName = userName
		return userName
	}
	var newRateLimiterEnv = func(limit string, customizer func(set *pflag.FlagSet)) {
		ctx = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
			Expect(set.Set("api.rate_limit", limit)).ToNot(HaveOccurred())
			Expect(set.Set("api.rate_limiting_enabled", "true")).ToNot(HaveOccurred())
			if customizer != nil {
				customizer(set)
			}
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

	BeforeEach(func() {
		newRateLimiterEnv("20-M,20-M", nil)
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

		Context("request is authorized", func() {

			When("basic auth (global Platform)", func() {
				It("no limit to be applied", func() {
					bulkRequest(ctx.SMWithBasic, osbURL+"/v2/catalog", 100)
				})
			})

			When("endpoint is public", func() {
				BeforeEach(func() {
					bulkRequest(ctx.SMWithOAuth, "/v1/info", 100)
				})
				It("no limit to be applied", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, "/v1/info")
				})
			})

			When("endpoint is excluded", func() {
				BeforeEach(func() {
					newRateLimiterEnv("20-M,20-M", func(set *pflag.FlagSet) {
						Expect(set.Set("api.rate_limit_exclude_paths", web.ServiceBrokersURL)).ToNot(HaveOccurred())
					})
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
				})
				It("no limit to be applied", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
			})

			When("doing too many requests", func() {
				BeforeEach(func() {
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
				})
				It("apply request limit", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
				It("reset the limit after", func() {
					time.Sleep(61 * time.Second)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
			})

			When("exclude client configured", func() {
				BeforeEach(func() {
					client := changeClientIdentifier()
					newRateLimiterEnv("20-M,20-M", func(set *pflag.FlagSet) {
						Expect(set.Set("api.rate_limit_exclude_clients", client)).ToNot(HaveOccurred())
					})
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 20)
				})
				It("no limit to be applied", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
			})

			When("two limiters configured", func() {
				BeforeEach(func() {
					newRateLimiterEnv("20-S,30-M", nil)
					changeClientIdentifier()
					// Exhaust seconds limiter, minute limiter drop from 30 to 10
					bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
					time.Sleep(1 * time.Second)
					// Exhaust minute limiter
					bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 10)
					// Expecting second limiter will reset, but minute should be still exhausted
					time.Sleep(3 * time.Second)
				})
				It("apply request limit using secondary limiter", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
				It("resets the limit after", func() {
					// Expecting all limiters will reset
					time.Sleep(61 * time.Second)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
			})

			When("custom path limiter configured", func() {
				BeforeEach(func() {
					newRateLimiterEnv("10-M", func(set *pflag.FlagSet) {
						Expect(set.Set("api.rate_limit_custom_paths", "5-M:"+web.PlatformsURL)).ToNot(HaveOccurred())
					})
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 5)
				})
				It("apply request limit using custom limiter", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
			})

			When("custom path limiter configured with multiple paths", func() {
				BeforeEach(func() {
					newRateLimiterEnv("10-M", func(set *pflag.FlagSet) {
						Expect(set.Set("api.rate_limit_custom_paths", "5-M:"+web.PlatformsURL+","+web.ServicePlansURL)).ToNot(HaveOccurred())
					})
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 5)
				})
				It("apply request limit", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
					// As it share same limiter with platforms
					expectLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)
				})
			})

			When("custom path limiter configured with multiple rates", func() {
				BeforeEach(func() {
					newRateLimiterEnv("100-M", func(set *pflag.FlagSet) {
						Expect(set.Set("api.rate_limit_custom_paths", "20-S,30-M:"+web.PlatformsURL)).ToNot(HaveOccurred())
					})
					changeClientIdentifier()
					// Exhaust seconds limiter, minute limiter drop from 30 to 10
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 20)
					time.Sleep(1 * time.Second)
					// Exhaust minute limiter
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 10)
					// Expecting second limiter will reset, but minute should be still exhausted
					time.Sleep(3 * time.Second)
				})
				It("apply request limit using secondary limiter", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
				It("resets the limit after", func() {
					// Expecting all limiters will reset
					time.Sleep(61 * time.Second)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
			})

			When("custom path limiter configured with independent paths", func() {
				BeforeEach(func() {
					newRateLimiterEnv("10-M", func(set *pflag.FlagSet) {
						Expect(set.Set("api.rate_limit_custom_paths", "5-M:"+web.PlatformsURL+";3-M:"+web.ServicePlansURL)).ToNot(HaveOccurred())
					})
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 5)
				})
				It("apply request limit (second independent limiter not impacted)", func() {
					// As platforms limiter is already exhausted
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
					// Exhausting plans limiter
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)
					expectLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)
				})
			})

		})
	})
})
