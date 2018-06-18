package osb_test

import (
	"encoding/json"
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
	"github.com/tidwall/sjson"
)

type object = common.Object
type array = common.Array

func TestPlugins(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "Plugin Tests Suite")
}

type testContext struct {
	SM                     *httpexpect.Expect
	smServer, brokerServer *httptest.Server
	brokerURL              string
}

func newTestContext(api *rest.API, brokerHandler http.Handler) *testContext {
	smServer := httptest.NewServer(common.GetServerRouter(api))
	SM := httpexpect.New(GinkgoT(), smServer.URL)

	common.RemoveAllBrokers(SM)
	brokerServer := httptest.NewServer(brokerHandler)
	brokerJSON := common.MakeBroker("broker1", brokerServer.URL, "")
	brokerID := common.RegisterBroker(brokerJSON, SM)
	brokerURL := "/v1/osb/" + brokerID

	return &testContext{
		SM:           SM,
		smServer:     smServer,
		brokerServer: brokerServer,
		brokerURL:    brokerURL,
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

var _ = Describe("Service Manager Plugins", func() {
	Describe("Modify catalog", func() {
		var ctx *testContext
		modifyRequestResponse := func(req *filter.Request, next filter.Handler) (*filter.Response, error) {
			var err error
			req.Body, err = sjson.SetBytes(req.Body, "extra", "request")
			if err != nil {
				return nil, err
			}
			req.Header.Del("content-length")

			res, err := next(req)
			if err != nil {
				return nil, err
			}

			res.Body, err = sjson.SetBytes(res.Body, "extra", "response")
			if err != nil {
				return nil, err
			}
			res.Header.Del("content-length")
			return res, nil
		}

		var statusCode int
		var responseBody []byte
		var brokerRequest *http.Request
		var brokerRequestBody *httpexpect.Value

		brokerHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			brokerRequest = req
			reqBody, err := ioutil.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}
			var reqData interface{}
			err = json.Unmarshal(reqBody, &reqData)
			if err != nil {
				panic(err)
			}
			brokerRequestBody = httpexpect.NewValue(GinkgoT(), reqData)

			rw.WriteHeader(statusCode)
			rw.Write(responseBody)
		})

		BeforeEach(func() {
			api := &rest.API{}
			api.RegisterPlugins(&TestPlugin{
				provision: modifyRequestResponse,
			})
			ctx = newTestContext(api, brokerHandler)
		})

		AfterEach(func() {
			ctx.Cleanup()
		})

		It("Plugin modifies the catalog", func() {
			provisionBody := object{
				"service_id": "s123",
				"plan_id":    "p123",
			}
			reply := ctx.SM.PUT(ctx.brokerURL + "/v2/service_instances/1234").
				WithJSON(provisionBody).
				Expect().Status(http.StatusOK).JSON().Object()

			reply.ValueEqual("extra", "response")
			brokerRequestBody.Object().Equal(object{
				"service_id": "s123",
				"plan_id":    "p123",
				"extra":      "request",
			})
		})
	})

})

type TestPlugin struct {
	fetchCatalog func(req *filter.Request, next filter.Handler) (*filter.Response, error)
	provision    func(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

func (p *TestPlugin) FetchCatalog(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	if p.fetchCatalog == nil {
		return next(req)
	}
	return p.fetchCatalog(req, next)

}

func (p *TestPlugin) Provision(req *filter.Request, next filter.Handler) (*filter.Response, error) {
	if p.provision == nil {
		return next(req)
	}
	return p.provision(req, next)
}
