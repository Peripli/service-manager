package filter_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

type object = common.Object
type array = common.Array

func TestFilters(t *testing.T) {
	os.Chdir("../..")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Tests Suite")
}

var _ = Describe("Service Manager Middlewares", func() {
	var ctx *common.TestContext
	var testBroker *common.Broker

	var testFilters []web.Filter

	JustBeforeEach(func() {
		api := &api.API{}
		api.RegisterFilters(testFilters...)
		ctx = common.NewTestContext(api)
		ctx.RegisterBroker("broker1", nil)
		testBroker = ctx.Brokers["broker1"]
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	Describe("Attach filter on multiple endpoints", func() {
		BeforeEach(func() {
			testFilters = []types.Filter{
				{
					Name: "OSB filter",
					RouteMatcher: types.RouteMatcher{
						PathPattern: "/v1/osb/**",
					},
					Middleware: func(req *types.Request, next types.SMHandler) (*types.Response, error) {
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
			ctx.SMWithBasic.GET(testBroker.OSBURL + "/v2/catalog").
				Expect().Status(http.StatusOK).Header("filter").Equal("called")

			ctx.SMWithBasic.PUT(testBroker.OSBURL+"/v2/service_instances/1234").
				WithHeader("Content-Type", "application/json").
				WithJSON(object{}).
				Expect().Status(http.StatusOK).Header("filter").Equal("called")

			ctx.SMWithBasic.DELETE(testBroker.OSBURL + "/v2/service_instances/1234/service_bindings/111").
				Expect().Status(http.StatusOK).Header("filter").Equal("called")

			ctx.SMWithOAuth.GET("/v1/service_brokers").
				Expect().Status(http.StatusOK).Header("filter").Empty()
		})
	})

	Describe("Attach filter on whole API", func() {
		var order string
		BeforeEach(func() {
			testFilters = []types.Filter{
				{
					Name: "Global filter",
					RouteMatcher: types.RouteMatcher{
						PathPattern: "/**",
					},
					Middleware: func(req *types.Request, next types.SMHandler) (*types.Response, error) {
						order += "a1"
						res, err := next(req)
						order += "a2"
						return res, err
					},
				},
				{
					Name: "/v1 filter",
					RouteMatcher: types.RouteMatcher{
						PathPattern: "/v1/**",
					},
					Middleware: func(req *types.Request, next types.SMHandler) (*types.Response, error) {
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
			ctx.SMWithOAuth.GET("/v1/platforms").
				Expect().Status(http.StatusOK)
			Expect(order).To(Equal("a1b1b2a2"))
		})
	})
})
