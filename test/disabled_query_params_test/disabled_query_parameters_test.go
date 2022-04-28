package disabled_query_params_test

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"net/http"
	"testing"
)

func TestDisabledQuery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Disable Query Parameters Tests Suite")
}

var _ = Describe("disable query parameter", func() {

	var (
		ctxBuilder *common.TestContextBuilder
		ctx        *common.TestContext
	)
	AfterSuite(func() {
		ctx.Cleanup()
	})
	Describe("query param is extended", func() {
		AfterEach(func() {
			ctx.Cleanup()
		})

		BeforeEach(func() {
			ctxBuilder = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
				Expect(set.Set("api.disabled_query_parameters", "")).ToNot(HaveOccurred())
			})
			ctx = ctxBuilder.Build()
		})

		Context("the query param is provided", func() {
			It("should succeed", func() {
				ctx.SMWithOAuth.GET(web.ServicePlansURL).WithQuery("environment", "cf").
					Expect().
					Status(http.StatusOK)
				ctx.SMWithOAuth.GET(web.ServiceOfferingsURL).WithQuery("environment", "cf").
					Expect().
					Status(http.StatusOK)
			})
		})

	})

	Describe("the query param is disabled", func() {
		AfterEach(func() {
			ctx.Cleanup()
		})

		BeforeEach(func() {
			ctxBuilder = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
				Expect(set.Set("api.disabled_query_parameters", "environment,someotherqueryparam")).ToNot(HaveOccurred())
			})
			ctx = ctxBuilder.Build()
		})

		Context("the query param is provided", func() {
			It("returns an error", func() {
				ctx.SMWithOAuth.GET(web.ServicePlansURL).WithQuery("environment", "cf").
					Expect().
					Status(http.StatusNotImplemented)
				ctx.SMWithOAuth.GET(web.ServicePlansURL).WithQuery("someotherqueryparam", "value").
					Expect().
					Status(http.StatusNotImplemented)
				ctx.SMWithOAuth.GET(web.ServiceOfferingsURL).WithQuery("environment", "cf").
					Expect().
					Status(http.StatusNotImplemented)
				ctx.SMWithOAuth.GET(web.ServiceOfferingsURL).WithQuery("someotherqueryparam", "value").
					Expect().
					Status(http.StatusNotImplemented)
			})
		})

		Context("query param is not in disabled query params settings", func() {
			It("succeed", func() {
				ctx.SMWithOAuth.GET(web.ServicePlansURL).WithQuery("check", "somevalue").
					Expect().
					Status(http.StatusOK)
				ctx.SMWithOAuth.GET(web.ServiceOfferingsURL).WithQuery("check", "somevalue").
					Expect().
					Status(http.StatusOK)
			})
		})
		Context("the query param is not provided", func() {
			It("should succeed", func() {
				ctx.SMWithOAuth.GET(web.ServicePlansURL).
					Expect().
					Status(http.StatusOK)
				ctx.SMWithOAuth.GET(web.ServiceOfferingsURL).
					Expect().
					Status(http.StatusOK)
			})
		})

	})
})
