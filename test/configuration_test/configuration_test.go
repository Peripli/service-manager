package configuration_test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"gopkg.in/square/go-jose.v2/json"

	"github.com/benjamintf1/unmarshalledmatchers"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

// TestConfiguration tests the Logging Config API
func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration API Tests Suite")
}

var _ = Describe("Service Manager Config API", func() {
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

	Context("GET", func() {
		It("returns the correct configuration that is used by the application", func() {
			expectedCfg := `{
				"api": {
					"client_id": "sm",
					"protected_labels": [],
					"skip_ssl_validation": false,
					"token_basic_auth": true
				},
				"file": {
					"format": "yml",
					"name": "application"
				},
				"health": {
					"monitored_platforms_threshold" : 10,
					"enable_platforms_indicator": false,
					"indicators": {
						"platforms": {
							"failures_threshold": 3,
							"fatal": true,
							"interval": "1m0s"
						},
						"storage": {
							"failures_threshold": 3,
							"fatal": true,
							"interval": "1m0s"
						}
					}
				},
				"httpclient": {
					"timeout": "4000ms",
					"dial_timeout": "4000ms",
					"idle_conn_timeout": "4000ms",
					"response_header_timeout": "4000ms",
					"skip_ssl_validation": true,
					"tls_handshake_timeout": "4000ms"
				},
				"log": {
					"format": "text",
					"level": "info",
					"output": "ginkgowriter"
				},
				"multitenancy": {
					"label_key": "tenant"
				},
				"operations": {
					"action_timeout": "15m0s",
            		"cleanup_interval": "1h0m0s",
					"default_pool_size": 20,
					"lifespan": "168h0m0s",
					"maintainer_retry_interval": "10m0s",
					"polling_interval": "1ms",
					"pools": "",
					"reconciliation_operation_timeout": "168h0m0s",
					"rescheduling_interval": "1ms",
					"sm_supported_platform_type": ["service-manager"]
				  },
				"server": {
					"host": "",
					"max_body_bytes": 1048576,
					"max_header_bytes": 1024,
					"port": 1234,
					"request_timeout": "30s",
					"shutdown_timeout": "4000ms"
				},
				"storage": {
					"encryption_key": "ejHjRNHbS0NaqARSRvnweVV9zcmhQEa8",
					"max_idle_connections": 5,
					"max_open_connections": 30,
					"notification": {
						"clean_interval": "24h",
						"keep_for": "24h",
						"max_reconnect_interval": "20s",
						"min_reconnect_interval": "200ms",
						"queues_size": 100
					},
					"skip_ssl_validation": false,
					"uri": "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
				},
				"websocket": {
					"ping_timeout": "4000ms",
					"write_timeout": "4000ms"
				}
			}`
			respBody := ctx.SMWithOAuth.GET(web.ConfigURL).
				Expect().
				Status(http.StatusOK).JSON().Object().Raw()
			bytes, err := json.Marshal(respBody)
			Expect(err).ToNot(HaveOccurred())
			Expect(bytes).To(unmarshalledmatchers.ContainUnorderedJSON(expectedCfg))
		})
	})

	Describe("Logging API", func() {
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
