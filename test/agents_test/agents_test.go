package agents_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestAgents(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agents API Tests Suite")
}


var _ = Describe("Agents API", func() {
	var (
		//ctxBuilder *common.TestContextBuilder
		//ctx        *common.TestContext
		//postHook   func(env.Environment, map[string]common.FakeServer)
	)

	BeforeEach(func() {
	//	ctxBuilder = common.NewTestContextBuilder()
		/*postHook := func(e env.Environment, servers map[string]common.FakeServer) {
			e.Set("api.token_basic_auth", tc.configBasicAuth)
		}*/
	//	ctx = common.NewTestContextBuilder().WithEnvPostExtensions(postHook).Build()
		//fmt.Println(ctx)

	})

	Context("when no versions are set", func() {
		It("should return an empty json object", func() {
		/*
			ctx.SM.GET(info.URL).
				Expect().
				Status(http.StatusOK).
				JSON().Object().Equal*/
		})

	})
})
