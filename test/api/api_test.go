package api

import (
	"net/http/httptest"
	"testing"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/server"
	_ "github.com/Peripli/service-manager/storage/postgres"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

func TestAPI(t *testing.T) {
	RunSpecs(t, "API Tests Suite")
}

var sm *httpexpect.Expect

var _ = Describe("Service Manager API", func() {
	var testServer *httptest.Server

	BeforeSuite(func() {
		srv, err := server.New(api.Default(), server.DefaultConfiguration())
		if err != nil {
			panic(err)
		}
		testServer = httptest.NewServer(srv.Router)
		sm = httpexpect.New(GinkgoT(), testServer.URL)
	})

	AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("Service Brokers", testBrokers)

	Describe("Platforms", testPlatforms)
})
