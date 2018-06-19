package filter_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type object = common.Object
type array = common.Array

func TestFilters(t *testing.T) {
	os.Chdir("../..")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Tests Suite")
}

var _ = Describe("Service Manager Filters", func() {
	var ctx *common.TestContext

	var testFilters []filter.Filter

	JustBeforeEach(func() {
		api := &rest.API{}
		api.RegisterFilters(testFilters...)
		ctx = common.NewTestContext(api)
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	Describe("Attach filter on multiple endpoints", func() {
		BeforeEach(func() {
			testFilters = []filter.Filter{
				{
					RequestMatcher: filter.RequestMatcher{
						PathPattern: "/v1/osb/**",
					},
					Middleware: func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
						res, err := next(req)
						if err == nil {
							res.Header.Set("filter", "called")
						}
						return res, err
					},
				},
			}
		})

		It("should be called only on OSB API", func() {
			ctx.SM.GET(ctx.OSBURL + "/v2/catalog").
				Expect().Status(http.StatusOK).Header("filter").Equal("called")

			ctx.SM.PUT(ctx.OSBURL+"/v2/service_instances/1234").
				WithHeader("Content-Type", "application/json").
				WithJSON(object{}).
				Expect().Status(http.StatusOK).Header("filter").Equal("called")

			ctx.SM.DELETE(ctx.OSBURL + "/v2/service_instances/1234/service_bindings/111").
				Expect().Status(http.StatusOK).Header("filter").Equal("called")

			ctx.SM.GET("/v1/service_brokers").
				Expect().Status(http.StatusOK).Header("filter").Empty()
		})
	})

	Describe("Attach filter on whole API", func() {
		var order string
		BeforeEach(func() {
			testFilters = []filter.Filter{
				{
					RequestMatcher: filter.RequestMatcher{
						PathPattern: "/v1/**",
					},
					Middleware: func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
						order += "a1"
						res, err := next(req)
						order += "a2"
						return res, err
					},
				},
				{
					RequestMatcher: filter.RequestMatcher{
						PathPattern: "/v1/**",
					},
					Middleware: func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
						order += "b1"
						res, err := next(req)
						order += "b2"
						return res, err
					},
				},
			}
		})

		It("should be called on platform API", func() {
			order = ""
			ctx.SM.GET("/v1/platforms").
				Expect().Status(http.StatusOK)
			Expect(order).To(Equal("a1b1b2a2"))
		})
	})
})
