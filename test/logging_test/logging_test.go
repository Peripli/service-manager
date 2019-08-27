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
		ctx                *common.TestContext
		initialLogSettings *log.Settings
	)

	BeforeSuite(func() {
		ctx = common.NewTestContextBuilder().Build()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	BeforeEach(func() {
		initialLogSettings = &log.Settings{
			Level:  "error",
			Format: "json",
			Output: "ginkgowriter",
		}
		_, err := log.Configure(context.TODO(), initialLogSettings)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("GET", func() {
		It("returns the correct logging configuration", func() {
			ctx.SMWithOAuth.GET(web.LoggingConfigURL).
				Expect().
				Status(http.StatusOK).JSON().Object().ContainsMap(map[string]string{
				"level":  initialLogSettings.Level,
				"format": initialLogSettings.Format,
			})
		})
	})

	Context("PUT", func() {
		When("the provided log level and log format are valid", func() {
			It("successfully modifies the logging configuration", func() {
				body := common.Object{
					"level":  "panic",
					"format": "text",
				}

				ctx.SMWithOAuth.PUT(web.LoggingConfigURL).
					WithJSON(body).Expect().Status(http.StatusOK)

				ctx.SMWithOAuth.GET(web.LoggingConfigURL).
					Expect().
					Status(http.StatusOK).JSON().Object().ContainsMap(map[string]interface{}{
					"level":  body["level"],
					"format": body["format"],
					"output": initialLogSettings.Output,
				})
			})
		})

		When("the provided log level is invalid", func() {
			It("returns 400 and does not modify the log configuration", func() {
				ctx.SMWithOAuth.PUT(web.LoggingConfigURL).
					WithJSON(common.Object{
						"level":  "invalid",
						"format": "text",
					}).Expect().Status(http.StatusBadRequest)

				ctx.SMWithOAuth.GET(web.LoggingConfigURL).
					Expect().
					Status(http.StatusOK).JSON().Object().Equal(initialLogSettings)
			})
		})

		When("the provided log format is invalid", func() {
			It("returns 400 and does not modify the log configuration", func() {
				ctx.SMWithOAuth.PUT(web.LoggingConfigURL).
					WithJSON(common.Object{
						"level":  "panic",
						"format": "invalid",
					}).Expect().Status(http.StatusBadRequest)

				ctx.SMWithOAuth.GET(web.LoggingConfigURL).
					Expect().
					Status(http.StatusOK).JSON().Object().Equal(initialLogSettings)
			})
		})

		When("log output is provided", func() {
			It("does not affect the actual logger output", func() {
				ctx.SMWithOAuth.PUT(web.LoggingConfigURL).
					WithJSON(common.Object{
						"output": os.Stderr.Name(),
					}).Expect().Status(http.StatusBadRequest)

				ctx.SMWithOAuth.GET(web.LoggingConfigURL).
					Expect().
					Status(http.StatusOK).JSON().Object().Equal(initialLogSettings)
			})
		})
	})
})
