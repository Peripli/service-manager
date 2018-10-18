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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/test/common"
	"github.com/pkg/errors"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"context"
	"time"

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

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func httpClient(reaction *common.HTTPReaction, checks *common.HTTPExpectations) *MockTransport {
	return &MockTransport{
		f: common.DoHTTP(reaction, checks),
	}
}

var _ = Describe("Client", func() {
	Describe("NewClient", func() {
		var settings *Settings

		BeforeEach(func() {
			settings = &Settings{
				User:              "admin",
				Password:          "admin",
				URL:               "http://example.com",
				OSBAPIPath:        "/osb",
				RequestTimeout:    5,
				ResyncPeriod:      5,
				SkipSSLValidation: false,
				Transport:         nil,
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
	const okResponse = `{
		"brokers": [
		{
			"id": "brokerID",
			"name": "brokerName",
			"description": "Service broker providing some valuable services",
			"created_at": "2016-06-08T16:41:26Z",
			"updated_at": "2016-06-08T16:41:26Z",
			"broker_url": "https://service-broker-url",
			"metadata": {

			},
			      "catalog":{  
         "services":[  
            {  
               "name":"fake-service",
               "id":"acb56d7c-XXXX-XXXX-XXXX-feb140a59a66",
               "description":"fake service",
               "tags":[  
                  "no-sql",
                  "relational"
               ],
               "requires":[  
                  "route_forwarding"
               ],
               "bindable":true,
               "metadata":{
                  "provider":{  
                     "name":"The name"
                  },
                  "listing":{  
                     "imageUrl":"http://example.com/cat.gif",
                     "blurb":"Add a blurb here",
                     "longDescription":"A long time ago, in a galaxy far far away..."
                  },
                  "displayName":"The Fake Broker"
               },
               "dashboard_client":{  
                  "id":"398e2f8e-XXXX-XXXX-XXXX-19a71ecbcf64",
                  "secret":"277cabb0-XXXX-XXXX-XXXX-7822c0a90e5d",
                  "redirect_uri":"http://localhost:1234"
               },
               "plan_updateable":true,
               "plans":[  
                  {  
                     "name":"fake-plan-1",
                     "id":"d3031751-XXXX-XXXX-XXXX-a42377d3320e",
                     "description":"Shared fake Server, 5tb persistent disk, 40 max concurrent connections",
                     "free":false,
                     "metadata":{  
                        "max_storage_tb":5,
                        "costs":[  
                           {  
                              "amount":{  
                                 "usd":99.0
                              },
                              "unit":"MONTHLY"
                           },
                           {  
                              "amount":{  
                                 "usd":0.99
                              },
                              "unit":"1GB of messages over 20GB"
                           }
                        ],
                        "bullets":[  
                           "Shared fake server",
                           "5 TB storage",
                           "40 concurrent connections"
                        ]
                     },
                     "schemas":{  
                        "service_instance":{  
                           "create":{  
                              "parameters":{  
                                 "$schema":"http://json-schema.org/draft-04/schema#",
                                 "type":"object",
                                 "properties":{  
                                    "billing-account":{  
                                       "description":"Billing account number used to charge use of shared fake server.",
                                       "type":"string"
                                    }
                                 }
                              }
                           },
                           "update":{  
                              "parameters":{  
                                 "$schema":"http://json-schema.org/draft-04/schema#",
                                 "type":"object",
                                 "properties":{  
                                    "billing-account":{  
                                       "description":"Billing account number used to charge use of shared fake server.",
                                       "type":"string"
                                    }
                                 }
                              }
                           }
                        },
                        "service_binding":{  
                           "create":{  
                              "parameters":{  
                                 "$schema":"http://json-schema.org/draft-04/schema#",
                                 "type":"object",
                                 "properties":{  
                                    "billing-account":{  
                                       "description":"Billing account number used to charge use of shared fake server.",
                                       "type":"string"
                                    }
                                 }
                              }
                           }
                        }
                     }
                  }
               ]
            }
         ]}
		}
	]}`

	catalogObject := func(brokers string) *osbc.CatalogResponse {
		c := &Brokers{}
		err := json.Unmarshal([]byte(brokers), c)
		if err != nil {
			panic(err)
		}
		return c.Brokers[0].Catalog
	}

	clientBrokersResponse := []Broker{
		{
			ID:        "brokerID",
			BrokerURL: "https://service-broker-url",
			Catalog:   catalogObject(okResponse),
			Metadata:  map[string]json.RawMessage{},
		},
	}

	type testCase struct {
		expectations *common.HTTPExpectations
		reaction     *common.HTTPReaction

		expectedErr      error
		expectedResponse []Broker
	}

	entries := []TableEntry{
		Entry("Successfully obtain brokers", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIInternalBrokers, "http://example.com"),
				Params: map[string]string{
					"catalog": "true",
				},
				Headers: map[string]string{
					"Authorization": "Basic " + basicAuth("admin", "admin"),
				},
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   okResponse,
				Err:    nil,
			},
			expectedResponse: clientBrokersResponse,
			expectedErr:      nil,
		}),

		Entry("Returns error when API returns error", testCase{
			expectations: &common.HTTPExpectations{
				URL: fmt.Sprintf(APIInternalBrokers, "http://example.com"),
				Params: map[string]string{
					"catalog": "true",
				},
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
				Params: map[string]string{
					"catalog": "true",
				},
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
				Params: map[string]string{
					"catalog": "true",
				},
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
		client, err := NewClient(&Settings{
			User:              "admin",
			Password:          "admin",
			URL:               "http://example.com",
			OSBAPIPath:        "/osb",
			RequestTimeout:    2 * time.Second,
			ResyncPeriod:      5 * time.Second,
			SkipSSLValidation: false,
			Transport:         httpClient(t.reaction, t.expectations),
		})
		Expect(err).ShouldNot(HaveOccurred())
		resp, err := client.GetBrokers(context.TODO())

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

	}, entries...)
})
