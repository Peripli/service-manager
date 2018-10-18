/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package app

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

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

func assertErrIsCFError(actualErr error, expectedErr CloudFoundryErr) {
	cause := errors.Cause(actualErr).(CloudFoundryErr)
	Expect(cause).To(MatchError(expectedErr))
}

func ccClient(URL string) (*ClientConfiguration, *PlatformClient) {
	cfConfig := &cfclient.Config{
		ApiAddress: URL,
	}
	regDetails := &RegistrationDetails{
		User:     "user",
		Password: "password",
	}
	config := &ClientConfiguration{
		Config:             cfConfig,
		CfClientCreateFunc: cfclient.NewClient,
		Reg:                regDetails,
	}
	client, err := NewClient(config)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(client).ShouldNot(BeNil())
	return config, client
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
			config *ClientConfiguration
		)

		BeforeEach(func() {
			config = &ClientConfiguration{
				Config:             cfclient.DefaultConfig(),
				CfClientCreateFunc: cfclient.NewClient,
				Reg: &RegistrationDetails{
					User:     "user",
					Password: "password",
				},
			}
		})

		Context("when create func fails", func() {
			BeforeEach(func() {
				config.CfClientCreateFunc = nil
			})

			It("returns an error", func() {
				_, err := NewClient(config)

				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when the config is invalid", func() {
			BeforeEach(func() {
				config.Config.ApiAddress = "invalidAPI"
			})

			It("returns an error", func() {
				_, err := NewClient(config)

				Expect(err).Should(HaveOccurred())
			})
		})
	})
})
