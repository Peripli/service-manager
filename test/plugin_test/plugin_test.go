package osb_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
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

type testContext struct {
	SM                     *httpexpect.Expect
	smServer, brokerServer *httptest.Server
	osbURL                 string
	broker                 *Broker
}

func newTestContext(api *rest.API) *testContext {
	smServer := httptest.NewServer(common.GetServerRouter(api))
	SM := httpexpect.New(GinkgoT(), smServer.URL)

	common.RemoveAllBrokers(SM)
	broker := &Broker{}
	brokerServer := httptest.NewServer(broker)
	brokerJSON := common.MakeBroker("broker1", brokerServer.URL, "")
	brokerID := common.RegisterBroker(brokerJSON, SM)
	osbURL := "/v1/osb/" + brokerID

	return &testContext{
		SM:           SM,
		smServer:     smServer,
		brokerServer: brokerServer,
		osbURL:       osbURL,
		broker:       broker,
	}
}

func (ctx *testContext) Cleanup() {
	if ctx == nil {
		return
	}
	if ctx.smServer != nil {
		common.RemoveAllBrokers(ctx.SM)
		ctx.smServer.Close()
	}
	if ctx.brokerServer != nil {
		ctx.brokerServer.Close()
	}
}

type Broker struct {
	StatusCode   int
	ResponseBody []byte
	Request      *http.Request
	RequestBody  *httpexpect.Value
}

func (b *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.Request = req

	if req.Method == http.MethodPatch || req.Method == http.MethodPost || req.Method == http.MethodPut {
		reqBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}
		var reqData interface{}
		err = json.Unmarshal(reqBody, &reqData)
		if err != nil {
			panic(err)
		}

		b.RequestBody = httpexpect.NewValue(GinkgoT(), reqData)
	}

	code := b.StatusCode
	if code == 0 {
		code = http.StatusOK
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)

	rw.Write(b.ResponseBody)
}

func (b *Broker) Called() bool {
	return b.Request != nil
}

var _ = Describe("Service Manager Plugins", func() {
	Describe("Modify request & response body", func() {
		var ctx *testContext
		var testPlugin TestPlugin

		BeforeEach(func() {
			testPlugin = TestPlugin{}
			api := &rest.API{}
			api.RegisterPlugins(testPlugin)
			ctx = newTestContext(api)
		})

		AfterEach(func() {
			ctx.Cleanup()
		})

		It("Plugin modifies the request & response body", func() {
			testPlugin["provision"] = func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
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

				return res, nil
			}
			ctx.broker.StatusCode = http.StatusCreated

			provisionBody := object{
				"service_id": "s123",
				"plan_id":    "p123",
			}
			reply := ctx.SM.PUT(ctx.osbURL + "/v2/service_instances/1234").
				WithJSON(provisionBody).Expect().Status(http.StatusCreated).JSON().Object()

			reply.ValueEqual("extra", "response")
			ctx.broker.RequestBody.Object().Equal(object{
				"service_id": "s123",
				"plan_id":    "p123",
				"extra":      "request",
			})
		})

		It("Plugin modifies the request & response headers", func() {
			testPlugin["fetchCatalog"] = func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
				h := req.Header.Get("extra")
				req.Header.Set("extra", h+"-request")

				res, err := next(req)
				if err != nil {
					return nil, err
				}

				res.Header.Set("extra", h+"-response")
				return res, nil
			}
			ctx.broker.StatusCode = http.StatusOK

			ctx.SM.GET(ctx.osbURL+"/v2/catalog").WithHeader("extra", "value").
				Expect().Status(http.StatusOK).Header("extra").Equal("value-response")

			Expect(ctx.broker.Request.Header.Get("extra")).To(Equal("value-request"))
		})

		It("Plugin aborts the request", func() {
			testPlugin["fetchCatalog"] = func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
				return nil, filter.NewErrorResponse(errors.New("Plugin error"), http.StatusBadRequest, "PluginErr")
			}

			ctx.SM.GET(ctx.osbURL + "/v2/catalog").
				Expect().Status(http.StatusBadRequest).JSON().Object().Equal(object{
				"error":       "PluginErr",
				"description": "Plugin error",
			})

			Expect(ctx.broker.Called()).To(BeFalse())
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
				testPlugin[op.name] = func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
					res, err := next(req)
					if err == nil {
						res.Header.Set("X-Plugin", op.name)
					}
					return res, err
				}

				ctx.SM.Request(op.method, ctx.osbURL+op.path).
					WithHeader("Content-Type", "application/json").
					WithJSON(object{}).
					Expect().Status(http.StatusOK).Header("X-Plugin").Equal(op.name)
			})
		}
	})

})

type TestPlugin map[string]func(req *filter.Request, next filter.Handler) (*filter.Response, error)

func (p TestPlugin) call(f filter.Middleware, req *filter.Request, next filter.Handler) (*filter.Response, error) {
	if f == nil {
		return next(req)
	}
	return f(req, next)
}

func (p TestPlugin) FetchCatalog(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["fetchCatalog"], req, next)
}

func (p TestPlugin) Provision(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["provision"], req, next)
}

func (p TestPlugin) Deprovision(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["deprovision"], req, next)
}

func (p TestPlugin) UpdateService(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["updateService"], req, next)
}

func (p TestPlugin) FetchService(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["fetchService"], req, next)
}

func (p TestPlugin) Bind(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["bind"], req, next)
}

func (p TestPlugin) Unbind(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["unbind"], req, next)
}

func (p TestPlugin) FetchBinding(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	return p.call(p["fetchBinding"], req, next)
}
