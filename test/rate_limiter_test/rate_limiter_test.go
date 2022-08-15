package filter_test

import (
	"context"
	"github.com/gofrs/uuid"
	"github.com/spf13/pflag"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/env"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/sm"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/test"
	"net/http"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/test/common"
)

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
			Expect(set.Set("api.rate_limit_exclude_paths", web.OperationsURL)).ToNot(HaveOccurred())
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
	var registerBroker = func() {
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
		broker := ctx.RegisterBrokerWithCatalog(catalog)
		brokerID := broker.Broker.ID
		common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, brokerID)

		username, password := test.RegisterBrokerPlatformCredentials(ctx.SMWithBasic, brokerID)
		ctx.SMWithBasic.SetBasicCredentials(ctx, username, password)
		osbURL = "/v1/osb/" + brokerID
	}

	BeforeEach(func() {
		newRateLimiterEnv("20-M", nil)
		registerBroker()
	})

	AfterEach(func() {
		ctx.Cleanup()
		filterContext.UserName = ""
	})

	Describe("rate limiter", func() {

		Context("request is authorized", func() {

			When("basic auth (global Platform)", func() {
				It("doesn't limit basic auth requests", func() {
					bulkRequest(ctx.SMWithBasic, osbURL+"/v2/catalog", 21)
				})
			})

			When("endpoint is public", func() {
				BeforeEach(func() {
					bulkRequest(ctx.SMWithOAuth, "/v1/info", 21)
				})
				It("doesn't limit public endpoints", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, "/v1/info")
				})
			})

			When("endpoint is excluded", func() {
				BeforeEach(func() {
					bulkRequest(ctx.SMWithOAuth, web.OperationsURL, 20)
				})
				It("doesn't limit excluded paths", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.OperationsURL)
				})
			})

			When("doing too many requests", func() {
				BeforeEach(func() {
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 20)
				})
				It("does limit", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
				It("reset the limit after timeout", func() {
					time.Sleep(61 * time.Second)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
			})

			When("exclude client configured", func() {
				BeforeEach(func() {
					client := changeClientIdentifier()
					newRateLimiterEnv("2-M,2-M", func(set *pflag.FlagSet) {
						Expect(set.Set("api.rate_limit_exclude_clients", client)).ToNot(HaveOccurred())
					})
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 2)
				})
				It("doesn't limit", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
			})

			When("two global limiters configured", func() {
				BeforeEach(func() {
					newRateLimiterEnv("2-S,3-M", nil)
					changeClientIdentifier()
					// Exhaust seconds limiter, minute limiter drop from 3 to 1
					bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 2)
					time.Sleep(1 * time.Second)
					// Exhaust minute limiter
					bulkRequest(ctx.SMWithOAuth, web.ServiceBrokersURL, 1)
					// Expecting second limiter will reset, but minute should be still exhausted
					time.Sleep(3 * time.Second)
				})
				It("does limit using secondary limiter", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
				It("resets the limit after timeout", func() {
					// Expecting all limiters will reset
					time.Sleep(61 * time.Second)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
				})
			})

			When("limiter for global and for specific path configured", func() {
				BeforeEach(func() {
					newRateLimiterEnv("10-M,5-M:"+web.PlatformsURL, nil)
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 5)
				})
				It("apply request limit on specific path", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
			})

			When("limiter for specific path configured", func() {
				BeforeEach(func() {
					newRateLimiterEnv("5-M:"+web.PlatformsURL, nil)
					changeClientIdentifier()
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 5)
				})
				It("apply request limit on specific path", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
				It("apply request limit on sub path too", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
				It("doesn't limit on other path", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)
				})
			})

			When("limiter for /v1/service_instances path and POST method is configured", func() {
				var servicePlanID string
				BeforeEach(func() {
					newRateLimiterEnv("6-M:"+web.ServiceInstancesURL+",5-M:"+web.ServiceInstancesURL+":POST", nil)
					registerBroker()
					servicePlanID = ctx.SMWithOAuth.List(web.ServicePlansURL).Element(0).Object().Value("id").String().Raw()
					changeClientIdentifier()
					for i := 1; i <= 5; i++ {
						requestBody := common.Object{
							"name":            "test-instance" + strconv.Itoa(i),
							"service_plan_id": servicePlanID,
						}
						ctx.SMWithOAuth.
							POST(web.ServiceInstancesURL).
							WithQuery("async", "false").
							WithJSON(requestBody).
							Expect().
							Status(http.StatusCreated)
					}
				})
				It("apply request limit on /v1/service_instances path and POST method", func() {
					requestBody := common.Object{
						"name":            "test-instance",
						"service_plan_id": servicePlanID,
					}
					ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
						WithQuery("async", "false").
						WithJSON(requestBody).
						Expect().
						Status(http.StatusTooManyRequests)
				})
				It("does not apply request limit on /v1/service_instances path GET method", func() {
					ctx.SMWithOAuth.GET(web.ServiceInstancesURL).
						Expect().
						Status(http.StatusOK)
				})
			})

			When("limiter for multiple paths configured", func() {
				BeforeEach(func() {
					newRateLimiterEnv("10-M,5-M:"+web.PlatformsURL+",2-M:"+web.ServicePlansURL, nil)
					changeClientIdentifier()
					// Counters: 10 (global), 5 (platforms), 2 (plans)
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 5) // 5,0,2
				})
				It("apply request limit", func() {
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL) // 4,0,2
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)         // 3,0,2
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)   // 2,0,1
					expectNonLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)   // 1,0,0
					expectLimitedRequest(ctx.SMWithOAuth, web.ServicePlansURL)      // 0,0,0
					expectLimitedRequest(ctx.SMWithOAuth, web.ServiceBrokersURL)    // 0,0,0
				})
			})

			When("limiter configured with global rate and multiple for specific path", func() {
				BeforeEach(func() {
					newRateLimiterEnv("100-M,2-S:"+web.PlatformsURL+",4-M:"+web.PlatformsURL, nil)
					changeClientIdentifier()
					// Exhaust seconds limiter, minute limiter drop from 4 to 2
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 2)
					time.Sleep(1 * time.Second)
					// Exhaust minute limiter
					bulkRequest(ctx.SMWithOAuth, web.PlatformsURL, 2)
					// Expecting second limiter will reset, but minute should be still exhausted
					time.Sleep(3 * time.Second)
				})
				It("apply request limit using secondary limiter", func() {
					expectLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
				XIt("limit expires after timeout", func() {
					// Expecting all limiters will reset
					time.Sleep(61 * time.Second)
					expectNonLimitedRequest(ctx.SMWithOAuth, web.PlatformsURL)
				})
			})
		})
	})
})
