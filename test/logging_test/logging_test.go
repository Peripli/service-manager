package log_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

// TestLogging tests the Logging Config API
func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OSB API Tests Suite")
}

var _ = Describe("Service Manager Logging Config API", func() {
	var (
		ctx *common.TestContext
	)

	BeforeSuite(func() {
		ctx = common.NewTestContextBuilder().Build()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	BeforeEach(func() {
		log.Configure(context.TODO(), &log.Settings{
			Level:  "error",
			Format: "json",
			Output: os.Stdout.Name(),
		})
	})

	Context("GET", func() {
		It("returns the correct logging configuration", func() {
			ctx.SMWithOAuth.GET(web.LoggingConfigURL).
				Expect().
				Status(http.StatusOK).JSON().Object().ContainsMap(map[string]string{
				"level":  "error",
				"format": "json",
			})
		})
	})

	Context("PUT", func() {
		It("successfully modifies the logging configuration", func() {
			ctx.SMWithOAuth.PUT(web.LoggingConfigURL).
				WithJSON(common.Object{
					"level":  "panic",
					"format": "text",
				}).Expect().Status(http.StatusOK)

			ctx.SMWithOAuth.GET(web.LoggingConfigURL).
				Expect().
				Status(http.StatusOK).JSON().Object().ContainsMap(map[string]string{
				"level":  "panic",
				"format": "text",
			})
		})
	})
})
