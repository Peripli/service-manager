package plugin_test

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/tidwall/sjson"
)

type object = common.Object
type array = common.Array

func TestPlugins(t *testing.T) {
	os.Chdir("../..")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Tests Suite")
}

var _ = Describe("Service Manager Plugins", func() {
	var ctx *common.TestContext

	AfterEach(func() {
		ctx.Cleanup()
	})

	Describe("Partial plugin", func() {
		BeforeEach(func() {
			api := &rest.API{}
			api.RegisterPlugins(&PartialPlugin{})
			ctx = common.NewTestContext(api)
		})

		It("should be called for provision and not for deprovision", func() {
			ctx.SM.PUT(ctx.OSBURL+"/v2/service_instances/1234").
				WithHeader("Content-Type", "application/json").
				WithJSON(object{}).
				Expect().Status(http.StatusOK).Header("X-Plugin").Equal("provision")
			ctx.SM.DELETE(ctx.OSBURL+"/v2/service_instances/1234").
				WithHeader("Content-Type", "application/json").
				WithJSON(object{}).
				Expect().Status(http.StatusOK).Header("X-Plugin").Empty()
		})
	})

	Describe("Complete plugin", func() {
		var testPlugin TestPlugin

		BeforeEach(func() {
			testPlugin = TestPlugin{}
			api := &rest.API{}
			api.RegisterPlugins(testPlugin)
			ctx = common.NewTestContext(api)
		})

		It("Plugin modifies the request & response body", func() {
			var resBodySize int
			testPlugin["provision"] = func(req *web.Request, next web.Handler) (*web.Response, error) {
				var err error
				req.Body, err = sjson.SetBytes(req.Body, "extra", "request")
				if err != nil {
					return nil, err
				}

				res, err := next(req)
				if err != nil {
					return nil, err
				}

				res.Body, err = sjson.SetBytes(res.Body, "extra", "response")
				if err != nil {
					return nil, err
				}
				resBodySize = len(res.Body)
				return res, nil
			}
			ctx.Broker.StatusCode = http.StatusCreated

			provisionBody := object{
				"service_id": "s123",
				"plan_id":    "p123",
			}
			resp := ctx.SM.PUT(ctx.OSBURL + "/v2/service_instances/1234").
				WithJSON(provisionBody).Expect().Status(http.StatusCreated)
			resp.Header("content-length").Equal(strconv.Itoa(resBodySize))
			reply := resp.JSON().Object()

			Expect(ctx.Broker.Request.Header.Get("content-length")).To(Equal(
				strconv.Itoa(len(ctx.Broker.RawRequestBody))))
			reply.ValueEqual("extra", "response")
			ctx.Broker.RequestBody.Object().Equal(object{
				"service_id": "s123",
				"plan_id":    "p123",
				"extra":      "request",
			})
		})

		It("Plugin modifies the request & response headers", func() {
			testPlugin["fetchCatalog"] = func(req *web.Request, next web.Handler) (*web.Response, error) {
				h := req.Header.Get("extra")
				req.Header.Set("extra", h+"-request")

				res, err := next(req)
				if err != nil {
					return nil, err
				}

				res.Header.Set("extra", h+"-response")
				return res, nil
			}
			ctx.Broker.StatusCode = http.StatusOK

			ctx.SM.GET(ctx.OSBURL+"/v2/catalog").WithHeader("extra", "value").
				Expect().Status(http.StatusOK).Header("extra").Equal("value-response")

			Expect(ctx.Broker.Request.Header.Get("extra")).To(Equal("value-request"))
		})

		It("Plugin aborts the request", func() {
			testPlugin["fetchCatalog"] = func(req *web.Request, next web.Handler) (*web.Response, error) {
				return nil, web.NewHTTPError(errors.New("Plugin error"), http.StatusBadRequest, "PluginErr")
			}

			ctx.SM.GET(ctx.OSBURL + "/v2/catalog").
				Expect().Status(http.StatusBadRequest).JSON().Object().Equal(object{
				"error":       "PluginErr",
				"description": "Plugin error",
			})

			Expect(ctx.Broker.Called()).To(BeFalse())
		})

		It("Request host header is properly set", func() {
			ctx.SM.GET(ctx.OSBURL + "/v2/catalog").
				Expect().Status(http.StatusOK)

			Expect(ctx.BrokerServer.URL).To(ContainSubstring(ctx.Broker.Request.Host))
		})

		osbOperations := []struct{ name, method, path string }{
			{"fetchCatalog", "GET", "/v2/catalog"},
			{"provision", "PUT", "/v2/service_instances/1234"},
			{"deprovision", "DELETE", "/v2/service_instances/1234"},
			{"updateService", "PATCH", "/v2/service_instances/1234"},
			{"fetchService", "GET", "/v2/service_instances/1234"},
			{"bind", "PUT", "/v2/service_instances/1234/service_bindings/111"},
			{"unbind", "DELETE", "/v2/service_instances/1234/service_bindings/111"},
			{"fetchBinding", "GET", "/v2/service_instances/1234/service_bindings/111"},
		}
		for _, op := range osbOperations {
			op := op
			It(fmt.Sprintf("Plugin intercepts %s operation", op.name), func() {
				testPlugin[op.name] = func(req *web.Request, next web.Handler) (*web.Response, error) {
					res, err := next(req)
					if err == nil {
						res.Header.Set("X-Plugin", op.name)
					}
					return res, err
				}

				ctx.SM.Request(op.method, ctx.OSBURL+op.path).
					WithHeader("Content-Type", "application/json").
					WithJSON(object{}).
					Expect().Status(http.StatusOK).Header("X-Plugin").Equal(op.name)
			})
		}

	})

})

type TestPlugin map[string]func(req *web.Request, next web.Handler) (*web.Response, error)

func (p TestPlugin) Name() string { return "TestPlugin" }

func (p TestPlugin) call(f web.Middleware, req *web.Request, next web.Handler) (*web.Response, error) {
	if f == nil {
		return next(req)
	}
	return f(req, next)
}

func (p TestPlugin) FetchCatalog(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["fetchCatalog"], req, next)
}

func (p TestPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["provision"], req, next)
}

func (p TestPlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["deprovision"], req, next)
}

func (p TestPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["updateService"], req, next)
}

func (p TestPlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["fetchService"], req, next)
}

func (p TestPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["bind"], req, next)
}

func (p TestPlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["unbind"], req, next)
}

func (p TestPlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.call(p["fetchBinding"], req, next)
}

type PartialPlugin struct{}

func (p PartialPlugin) Name() string { return "PartialPlugin" }

func (p PartialPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	res, err := next(req)
	if err == nil {
		res.Header.Set("X-Plugin", "provision")
	}
	return res, err
}
