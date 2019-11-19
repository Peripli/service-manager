package cf_test

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/Peripli/service-manager/pkg/agent"

	"github.com/Peripli/service-manager/cmd/cf-agent/cf"
	"github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/pkg/errors"
)

const InvalidJSON = `{invalidjson`

type expectedRequest struct {
	Method   string
	Path     string
	RawQuery string
	Headers  map[string][]string
	Body     interface{}
}

type reactionResponse struct {
	Code    int
	Body    interface{}
	Error   error
	Headers map[string][]string
}

type mockRoute struct {
	requestChecks expectedRequest
	reaction      reactionResponse
}

func appendRoutes(server *ghttp.Server, routes ...*mockRoute) {
	for _, route := range routes {
		var handlers []http.HandlerFunc

		if route == nil || reflect.DeepEqual(*route, mockRoute{}) {
			continue
		}

		if route.requestChecks.RawQuery != "" {
			handlers = append(handlers, ghttp.VerifyRequest(route.requestChecks.Method, route.requestChecks.Path, route.requestChecks.RawQuery))
		} else {
			handlers = append(handlers, ghttp.VerifyRequest(route.requestChecks.Method, route.requestChecks.Path))
		}

		if route.requestChecks.Body != nil {
			handlers = append(handlers, ghttp.VerifyJSONRepresenting(route.requestChecks.Body))
		}

		for key, values := range route.requestChecks.Headers {
			handlers = append(handlers, ghttp.VerifyHeaderKV(key, values...))
		}

		if route.reaction.Error != nil {
			handlers = append(handlers, ghttp.RespondWithJSONEncodedPtr(&route.reaction.Code, &route.reaction.Error))

		} else {
			handlers = append(handlers, ghttp.RespondWithJSONEncodedPtr(&route.reaction.Code, &route.reaction.Body))
		}

		server.AppendHandlers(ghttp.CombineHandlers(handlers...))
	}
}

func encodeQuery(query string) string {
	q := url.Values{}
	q.Set("q", query)
	return q.Encode()
}

// can directly use this to verify if already defined routes were hit x times
func verifyRouteHits(server *ghttp.Server, expectedHitsCount int, route *mockRoute) {
	var hitsCount int
	expected := route.requestChecks
	for _, r := range server.ReceivedRequests() {
		methodsMatch := r.Method == expected.Method
		pathsMatch := r.URL.Path == expected.Path
		values, err := url.ParseQuery(expected.RawQuery)
		Expect(err).ShouldNot(HaveOccurred())
		queriesMatch := reflect.DeepEqual(r.URL.Query(), values)

		if methodsMatch && pathsMatch && queriesMatch {
			hitsCount++
		}
	}

	if expectedHitsCount != hitsCount {
		Fail(fmt.Sprintf("Request with method = %s, path = %s, rawQuery = %s expected to be received %d "+
			"times but was received %d times", expected.Method, expected.Path, expected.RawQuery, expectedHitsCount, hitsCount))
	}
}

func verifyReqReceived(server *ghttp.Server, times int, method, path string, rawQuery ...string) {
	timesReceived := 0
	for _, req := range server.ReceivedRequests() {
		if req.Method == method && strings.Contains(req.URL.Path, path) {
			if len(rawQuery) == 0 {
				timesReceived++
				continue
			}
			values, err := url.ParseQuery(rawQuery[0])
			Expect(err).ShouldNot(HaveOccurred())
			if reflect.DeepEqual(req.URL.Query(), values) {
				timesReceived++
			}
		}
	}
	if times != timesReceived {
		Fail(fmt.Sprintf("Request with method = %s, path = %s, rawQuery = %s expected to be received %d "+
			"times but was received %d times", method, path, rawQuery, times, timesReceived))
	}
}

func assertErrCauseIsCFError(actualErr error, expectedErr cf.CloudFoundryErr) {
	cause := errors.Cause(actualErr).(cf.CloudFoundryErr)
	Expect(cause).To(MatchError(expectedErr))
}

func ccClient(URL string) (*cf.Settings, *cf.PlatformClient) {
	return ccClientWithThrottling(URL, 50)
}

func ccClientWithThrottling(URL string, maxAllowedParallelRequests int) (*cf.Settings, *cf.PlatformClient) {
	cfConfig := &cfclient.Config{
		ApiAddress: URL,
	}
	config := &cf.ClientConfiguration{
		Config:           cfConfig,
		CFClientProvider: cfclient.NewClient,
	}
	settings := &cf.Settings{
		Settings: *agent.DefaultSettings(),
		CF:       config,
	}
	settings.Reconcile.URL = "http://10.0.2.2"
	settings.Reconcile.MaxParallelRequests = maxAllowedParallelRequests
	settings.Reconcile.LegacyURL = "http://proxy.com"
	settings.Sm.URL = "http://10.0.2.2"
	settings.Sm.User = "user"
	settings.Sm.Password = "password"

	client, err := cf.NewClient(settings)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(client).ShouldNot(BeNil())
	return settings, client
}

func fakeCCServer(allowUnhandled bool) *ghttp.Server {
	ccServer := ghttp.NewServer()
	v2InfoResponse := fmt.Sprintf(`
										{
											"api_version":"%[1]s",
											"authorization_endpoint": "%[2]s",
											"token_endpoint": "%[2]s",
											"login_endpoint": "%[2]s"
										}`,
		"2.5", ccServer.URL())
	ccServer.RouteToHandler(http.MethodGet, "/v2/info", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(v2InfoResponse))
	})
	ccServer.RouteToHandler(http.MethodPost, "/oauth/token", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(`
						{
							"token_type":    "bearer",
							"access_token":  "access",
							"refresh_token": "refresh",
							"expires_in":    "123456"
						}`))
	})
	ccServer.AllowUnhandledRequests = allowUnhandled
	return ccServer
}

var _ = Describe("Client", func() {
	Describe("NewClient", func() {
		var (
			settings *cf.Settings
		)

		BeforeEach(func() {
			config := &cf.ClientConfiguration{
				Config:           cfclient.DefaultConfig(),
				CFClientProvider: cfclient.NewClient,
			}
			settings = &cf.Settings{
				Settings: *agent.DefaultSettings(),
				CF:       config,
			}

			settings.Reconcile.URL = "http://10.0.2.2"
		})

		Context("when create func fails", func() {
			BeforeEach(func() {
				settings.CF.CFClientProvider = nil
			})

			It("returns an error", func() {
				_, err := cf.NewClient(settings)

				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when the config is invalid", func() {
			BeforeEach(func() {
				settings.CF.Config.ApiAddress = "invalidAPI"
			})

			It("returns an error", func() {
				_, err := cf.NewClient(settings)

				Expect(err).Should(HaveOccurred())
			})
		})
	})
})
