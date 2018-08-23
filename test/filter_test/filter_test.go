package filter_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/Peripli/service-manager/pkg/web"
)

type object = common.Object

func TestFilters(t *testing.T) {
	os.Chdir("../..")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filter Tests Suite")
}

var _ = Describe("Service Manager Filters", func() {
	var ctx *common.TestContext
	var testBroker *common.Broker

	var testFilters []web.Filter
	var order string

	JustBeforeEach(func() {
		api := &web.API{}
		api.RegisterFilters(testFilters...)
		ctx = common.NewTestContextFromAPIs(nil, api)
		testBroker = ctx.RegisterBroker("broker1", nil)
		order = ""
	})

	AfterEach(func() {
		ctx.Cleanup()
	})

	Describe("Attach filter on multiple endpoints", func() {
		BeforeEach(func() {
			testFilters = []web.Filter{
				osbTestFilter{state: &order},
			}
		})

		Context("should be called only on OSB API", func() {
			Specify("/v2/catalog", func() {
				ctx.SMWithBasic.GET(testBroker.OSBURL + "/v2/catalog").
					Expect().Status(http.StatusOK)
				Expect(order).To(Equal("osb1osb2"))
			})

			Specify("/v2/service_instances/1234", func() {
				ctx.SMWithBasic.PUT(testBroker.OSBURL+"/v2/service_instances/1234").
					WithHeader("Content-Type", "application/json").
					WithJSON(object{}).
					Expect().Status(http.StatusOK)
				Expect(order).To(Equal("osb1osb2"))

			})

			Specify("/v2/service_instances/1234/service_bindings/111", func() {
				ctx.SMWithBasic.DELETE(testBroker.OSBURL + "/v2/service_instances/1234/service_bindings/111").
					Expect().Status(http.StatusOK)
				Expect(order).To(Equal("osb1osb2"))
			})

			Specify("/v1/service_brokers", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").
					Expect().Status(http.StatusOK)
				Expect(order).ToNot(Equal("osb1osb2"))
			})

			Specify("/v1/platforms", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").
					Expect().Status(http.StatusOK)
				Expect(order).ToNot(Equal("osb1osb2"))
			})
		})
	})

	Describe("Attach filter on whole API", func() {
		BeforeEach(func() {
			testFilters = []web.Filter{
				globalTestFilterA{state: &order},
				globalTestFilterB{state: &order},
			}
		})

		It("should be called on platform API", func() {
			ctx.SMWithOAuth.GET("/v1/platforms").
				Expect().Status(http.StatusOK)
			Expect(order).To(Equal("a1b1b2a2"))
		})
	})
})

type osbTestFilter struct {
	state *string
}

func (tf osbTestFilter) Name() string {
	return "OSB Filter"
}

func (tf osbTestFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/v1/osb/**"),
			},
		},
	}
}

func (tf osbTestFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
		*tf.state += "osb1"
		res, err := next.Handle(request)
		if err == nil {
			*tf.state += "osb2"
		}
		return res, err
}

type globalTestFilterA struct {
	state *string
}

func (gfa globalTestFilterA) Name() string {
	return "GlobalTestFilterA"
}

func (gfa globalTestFilterA) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
			},
		},
	}
}

func (gfa globalTestFilterA) Run(request *web.Request, next web.Handler) (*web.Response,error) {
		*gfa.state += "a1"
		res, err := next.Handle(request)
		if err == nil {
			*gfa.state += "a2"
		}
		return res, err
}

type globalTestFilterB struct {
	state *string
}

func (gfb globalTestFilterB) Name() string {
	return "GlobalTestFilterB"
}

func (gfb globalTestFilterB) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/v1/**"),
			},
		},
	}
}

func (gfb globalTestFilterB) Run(request *web.Request, next web.Handler) (*web.Response,error) {
		*gfb.state += "b1"
		res, err := next.Handle(request)
		if err == nil {
			*gfb.state += "b2"
		}
		return res, err
}

