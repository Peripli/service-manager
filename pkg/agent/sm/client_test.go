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

package sm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/pkg/errors"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type MockTransport struct {
	f func(req *http.Request) (*http.Response, error)
}

func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.f(req)
}

func httpClient(reaction *common.HTTPReaction, checks *common.HTTPExpectations) *MockTransport {
	return &MockTransport{
		f: common.DoHTTP(reaction, checks),
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

var _ = Describe("Client", func() {
	Describe("NewClient", func() {
		var settings *Settings

		BeforeEach(func() {
			settings = &Settings{
				User:                 "admin",
				Password:             "admin",
				URL:                  "http://example.com",
				OSBAPIPath:           "/osb",
				NotificationsAPIPath: "/v1/notifications",
				RequestTimeout:       5,
				SkipSSLValidation:    false,
				Transport:            nil,
			}
		})

		Context("when config is invalid", func() {
			It("returns an error", func() {
				settings.User = ""
				_, err := NewClient(settings)

				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when config is valid", func() {
			Context("when transport is present in config", func() {
				It("it uses it as base transport", func() {
					settings.Transport = http.DefaultTransport

					client, err := NewClient(settings)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(client.httpClient.Transport.(*BasicAuthTransport).Rt).To(Equal(http.DefaultTransport))

				})
			})

			Context("when transport is not present in config", func() {
				It("uses a skip ssl transport as base transport", func() {
					client, err := NewClient(settings)
					Expect(err).ShouldNot(HaveOccurred())
					transport := client.httpClient.Transport.(*BasicAuthTransport)
					_, ok := transport.Rt.(*SkipSSLTransport)
					Expect(ok).To(BeTrue())
				})
			})
		})
	})

	type testCase struct {
		expectations *common.HTTPExpectations
		reaction     *common.HTTPReaction

		expectedErr      error
		expectedResponse interface{}
	}

	newClient := func(t *testCase) *ServiceManagerClient {
		client, err := NewClient(&Settings{
			User:                 "admin",
			Password:             "admin",
			URL:                  "http://example.com",
			OSBAPIPath:           "/osb",
			NotificationsAPIPath: "/v1/notifications",
			RequestTimeout:       2 * time.Second,
			SkipSSLValidation:    false,
			Transport:            httpClient(t.reaction, t.expectations),
		})
		Expect(err).ShouldNot(HaveOccurred())
		return client
	}

	assertResponse := func(t *testCase, resp interface{}, err error) {
		if t.expectedErr != nil {
			Expect(errors.Cause(err).Error()).To(ContainSubstring(t.expectedErr.Error()))
		} else {
			Expect(err).To(BeNil())
		}

		if t.expectedResponse != nil {
			Expect(resp).To(Equal(t.expectedResponse))
		} else {
			Expect(resp).To(BeNil())
		}
	}

	const okBrokerResponse = `{
		"service_brokers": [
		{
			"id": "brokerID",
			"name": "brokerName",
			"description": "Service broker providing some valuable services",
			"broker_url": "https://service-broker-url"
		}
		]
	}`

	clientBrokersResponse := []*types.ServiceBroker{
		{
			Base: types.Base{
				ID: "brokerID",
			},
			Description: "Service broker providing some valuable services",
			Name:        "brokerName",
			BrokerURL:   "https://service-broker-url",
		},
	}

	brokerEntries := []TableEntry{
		Entry("Successfully obtain brokers", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIInternalBrokers, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   okBrokerResponse,
				Err:    nil,
			},
			expectedResponse: clientBrokersResponse,
			expectedErr:      nil,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIInternalBrokers, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Err:    fmt.Errorf("error"),
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("error"),
		}),

		Entry("Returns error when API response body is invalid", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIInternalBrokers, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   `invalid`,
				Err:    nil,
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("Failed to decode request body"),
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIInternalBrokers, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Body:   `{"error":"error"}`,
				Err:    nil,
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("StatusCode: 500 Body: {\"error\":\"error\"}"),
		}),
	}

	DescribeTable("GETBrokers", func(t testCase) {
		client := newClient(&t)
		resp, err := client.GetBrokers(context.TODO())
		assertResponse(&t, resp, err)
	}, brokerEntries...)

	const okPlanResponse = `{
		"service_plans": [
			 {
				  "created_at": "2018-12-27T09:14:54Z",
				  "updated_at": "2018-12-27T09:14:54Z",
				  "id": "180dd7fb-1c6e-41fe-95ee-aefb51513032",
				  "name": "dummy1plan1",
				  "description": "dummy 1 example plan 1",
				  "catalog_id": "1f400825-1434-5278-9913-dfcf63fcd647",
				  "catalog_name": "dummy1plan1",
				  "free": false,
				  "bindable": true,
				  "plan_updateable": false,
				  "service_offering_id": "47c7790a-3cd1-4520-a030-471f91dc616e"
			 }
		]
	}`

	servicePlans := func(servicePlans string) []*types.ServicePlan {
		c := make(map[string][]*types.ServicePlan)
		err := json.Unmarshal([]byte(servicePlans), &c)
		if err != nil {
			panic(err)
		}
		if c["service_plans"] == nil {
			panic("could not unmarshal service plans")
		}
		return c["service_plans"]
	}

	planEntries := []TableEntry{
		Entry("Successfully obtain plans", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIPlans, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   okPlanResponse,
				Err:    nil,
			},
			expectedResponse: servicePlans(okPlanResponse),
			expectedErr:      nil,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIPlans, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Body:   okPlanResponse,
				Err:    fmt.Errorf("expected error"),
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("expected error"),
		}),
	}

	DescribeTable("GETPlans", func(t testCase) {
		client := newClient(&t)
		resp, err := client.GetPlans(context.TODO())
		assertResponse(&t, resp, err)
	}, planEntries...)

	const okVisibilityResponse = `{
		"visibilities": [
			 {
				  "id": "127b5b3a-c0bc-45be-bcaf-f1083566214f",
				  "platform_id": "bf092091-76ba-4398-a301-40472b794aea",
				  "service_plan_id": "180dd7fb-1c6e-41fe-95ee-aefb51513032",
				  "labels": {
						"organization_guid": [
							"d0761213-012d-4bc5-8a7b-7780875d8913",
							"15317fc3-693c-423a-90ba-6f86d6559abe"
						],
						"something": ["generic"]
				  },
				  "created_at": "2018-12-27T14:35:23Z",
				  "updated_at": "2018-12-27T14:35:23Z"
			 }
		]
   }`

	serviceVisibilities := func(serviceVisibilities string) []*types.Visibility {
		c := types.Visibilities{}
		err := json.Unmarshal([]byte(serviceVisibilities), &c)
		if err != nil {
			panic(err)
		}
		return c.Visibilities
	}

	visibilitiesEntries := []TableEntry{
		Entry("Successfully obtain visibilities", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIVisibilities, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   okVisibilityResponse,
				Err:    nil,
			},
			expectedResponse: serviceVisibilities(okVisibilityResponse),
			expectedErr:      nil,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIVisibilities, "http://example.com"),
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Body:   okPlanResponse,
				Err:    fmt.Errorf("expected error"),
			},
			expectedResponse: nil,
			expectedErr:      fmt.Errorf("expected error"),
		}),
	}

	DescribeTable("GETPlans", func(t testCase) {
		client := newClient(&t)
		resp, err := client.GetVisibilities(context.TODO())
		assertResponse(&t, resp, err)
	}, visibilitiesEntries...)
})
