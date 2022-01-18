package agents_test

import (
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"testing"
)

func TestAgents(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agents API Tests Suite")
}

var _ = Describe("agents API", func() {
	var (
		ctxBuilder *common.TestContextBuilder
		ctx        *common.TestContext
		postHook   func(env.Environment, map[string]common.FakeServer)
	)

	BeforeEach(func() {
		ctxBuilder = common.NewTestContextBuilder()

	})

	Context("when no versions are set", func() {
		BeforeEach(func() {
			ctx = ctxBuilder.Build()
		})
		It("should return an empty json object", func() {
			ctx.SM.GET(web.AgentsURL).
				Expect().
				Status(http.StatusOK).
				JSON().Object().Equal(BeEmpty())
		})
	})

	Context("when versions are set", func() {
		BeforeEach(func() {
			postHook = func(e env.Environment, servers map[string]common.FakeServer) {
				e.Set("agents.versions", `{"cf-versions":["1.0.0", "1.0.1", "1.0.2"],"k8s-versions":["2.0.0", "2.0.1"]}`)
			}
			ctx = ctxBuilder.WithEnvPostExtensions(postHook).Build()
		})
		It("should return supported veresions", func() {
			jsonResponse := ctx.SM.GET(web.AgentsURL).
				Expect().
				Status(http.StatusOK).
				JSON().Object()
			jsonResponse.Value("cf-versions").Array().Length().Equal(3)
			jsonResponse.Value("cf-versions").Array().First().String().Equal("1.0.0")
			jsonResponse.Value("cf-versions").Array().Last().String().Equal("1.0.2")
			jsonResponse.Value("k8s-versions").Array().Length().Equal(2)
			jsonResponse.Value("k8s-versions").Array().First().String().Equal("2.0.0")
			jsonResponse.Value("k8s-versions").Array().Last().String().Equal("2.0.1")
		})
	})
})
