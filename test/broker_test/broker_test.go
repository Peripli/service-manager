/*
*    Copyright 2018 The Service Manager Authors
*
*    Licensed under the Apache License, Version 2.0 (the "License");
*    you may not use this file except in compliance with the License.
*    You may obtain a copy of the License at
*
*        http://www.apache.org/licenses/LICENSE-2.0
*
*    Unless required by applicable law or agreed to in writing, software
*    distributed under the License is distributed on an "AS IS" BASIS,
*    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*    See the License for the specific language governing permissions and
*    limitations under the License.
 */
package broker_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/test/tls_settings"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/httpclient"
	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/test"
	"github.com/gavv/httpexpect"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cast"
)

func TestBrokers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceBroker API Tests Suite")
}

const TenantLabelKey = "tenant"

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.ServiceBrokersURL,
	SupportedOps: []test.Op{
		test.Get, test.List, test.Delete, test.DeleteList, test.Patch,
	},
	ResourceType: types.ServiceBrokerType,
	MultitenancySettings: &test.MultitenancySettings{
		ClientID:           "tenancyClient",
		ClientIDTokenClaim: "cid",
		TenantTokenClaim:   "zid",
		LabelKey:           TenantLabelKey,
		TokenClaims: map[string]interface{}{
			"cid": "tenancyClient",
			"zid": "tenantID",
		},
	},
	SupportsAsyncOperations:                true,
	ResourceBlueprint:                      blueprint(true),
	ResourceWithoutNullableFieldsBlueprint: blueprint(false),
	ResourcePropertiesToIgnore:             []string{"last_operation"},
	PatchResource:                          test.APIResourcePatch,
	AdditionalTests: func(ctx *TestContext, t *test.TestCase) {
		Context("additional non-generic tests", func() {
			var (
				brokerServer                                           *BrokerServer
				brokerWithLabelsServer                                 *BrokerServer
				brokerServerWithBrokerCertificate                      *BrokerServer
				brokerServerWithSMCertficate                           *BrokerServer
				postBrokerRequestWithNoLabels                          Object
				expectedBrokerResponse                                 Object
				postBrokerRequestWithTLS                               Object
				postBrokerRequestWithTLSandBasic                       Object
				expectedBrokerResponseTLS                              Object
				expectedBrokerServerWithServiceManagerMtlsAndBasicAuth Object
				postBrokerRequestWithTLSToWrongBrokerServer            Object
				postBrokerRequestWithTLSNoCert                         Object
				labels                                                 Object
				postBrokerBasicServerMtls                              Object
				postBrokerRequestWithLabels                            labeledBroker

				repository storage.Repository
			)

			assertInvocationCount := func(requests []*http.Request, invocationCount int) {
				Expect(len(requests)).To(Equal(invocationCount))
			}

			AfterEach(func() {
				if brokerServer != nil {
					brokerServer.Close()
				}

				if brokerWithLabelsServer != nil {
					brokerWithLabelsServer.Close()
				}

				if brokerServerWithBrokerCertificate != nil {
					brokerServerWithBrokerCertificate.Close()
				}

				if brokerServerWithSMCertficate != nil {
					brokerServerWithSMCertficate.Close()
				}
			})

			BeforeEach(func() {
				brokerServer = NewBrokerServer()
				brokerWithLabelsServer = NewBrokerServer()
				brokerServerWithBrokerCertificate = NewBrokerServerMTLS([]byte(tls_settings.BrokerCertificate), []byte(tls_settings.BrokerCertificateKey),
					[]byte(tls_settings.ClientCaCertificate))
				brokerServerWithSMCertficate = NewBrokerServerMTLS([]byte(tls_settings.BrokerCertificate), []byte(tls_settings.BrokerCertificateKey),
					[]byte(tls_settings.SMRootCaCertificate))
				brokerServerWithSMCertficate.Reset()
				brokerServerWithBrokerCertificate.Reset()
				brokerServer.Reset()
				brokerWithLabelsServer.Reset()
				brokerName := "brokerName"
				brokerNameWithTLS := "brokerNameTLS"
				brokerWithLabelsName := "brokerWithLabelsName"
				brokerDescription := "description"
				brokerWithLabelsDescription := "descriptionWithLabels"

				brokerServerWithServiceManagerMtlsName := "brokerServerWithServiceManagerMtlsName"
				postBrokerRequestWithNoLabels = Object{
					"name":        brokerName,
					"broker_url":  brokerServer.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"basic": Object{
							"username": brokerServer.Username,
							"password": brokerServer.Password,
						},
					},
				}
				expectedBrokerResponse = Object{
					"name":        brokerName,
					"broker_url":  brokerServer.URL(),
					"description": brokerDescription,
				}

				labels = Object{
					"cluster_id": Array{"cluster_id_value"},
					"org_id":     Array{"org_id_value1", "org_id_value2", "org_id_value3"},
				}

				postBrokerRequestWithLabels = Object{
					"name":        brokerWithLabelsName,
					"broker_url":  brokerWithLabelsServer.URL(),
					"description": brokerWithLabelsDescription,
					"credentials": Object{
						"basic": Object{
							"username": brokerWithLabelsServer.Username,
							"password": brokerWithLabelsServer.Password,
						},
					},
					"labels": labels,
				}

				postBrokerBasicServerMtls = Object{
					"name":        brokerServerWithServiceManagerMtlsName,
					"broker_url":  brokerServerWithSMCertficate.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"sm_provided_tls_credentials": true,
					},
					"labels": labels,
				}

				expectedBrokerServerWithServiceManagerMtlsAndBasicAuth = Object{
					"name":        brokerServerWithServiceManagerMtlsName,
					"broker_url":  brokerServerWithSMCertficate.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"sm_provided_tls_credentials": true,
						"basic": Object{
							"username": brokerServerWithSMCertficate.Username,
							"password": brokerServerWithSMCertficate.Password,
						},
					},
				}

				postBrokerRequestWithTLSToWrongBrokerServer = Object{
					"name":        "wrong-broker-tls",
					"broker_url":  brokerServerWithSMCertficate.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"tls": Object{
							"client_certificate": tls_settings.ClientCertificate,
							"client_key":         tls_settings.ClientKey,
						},
					},
				}

				postBrokerRequestWithTLS = Object{
					"name":        brokerNameWithTLS,
					"broker_url":  brokerServerWithBrokerCertificate.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"tls": Object{
							"client_certificate": tls_settings.ClientCertificate,
							"client_key":         tls_settings.ClientKey,
						},
					},
				}

				postBrokerRequestWithTLSNoCert = Object{
					"name":        brokerNameWithTLS,
					"broker_url":  brokerServerWithBrokerCertificate.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"basic": Object{
							"username": brokerServer.Username,
							"password": brokerServer.Password,
						},
					},
				}

				postBrokerRequestWithTLSandBasic = Object{
					"name":        brokerNameWithTLS,
					"broker_url":  brokerServerWithBrokerCertificate.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"basic": Object{
							"username": brokerServer.Username,
							"password": brokerServer.Password,
						},
						"tls": Object{
							"client_certificate": tls_settings.ClientCertificate,
							"client_key":         tls_settings.ClientKey,
						},
					},
				}

				expectedBrokerResponseTLS = Object{
					"name":        brokerNameWithTLS,
					"broker_url":  brokerServerWithBrokerCertificate.URL(),
					"description": brokerDescription,
					"credentials": Object{
						"basic": Object{
							"username": brokerServer.Username,
							"password": brokerServer.Password,
						},
						"tls": Object{
							"client_certificate": tls_settings.ClientCertificate,
							"client_key":         tls_settings.ClientKey,
						},
					},
				}

				RemoveAllBrokers(ctx.SMRepository)

				repository = ctx.SMRepository
			})

			Describe("GET", func() {
				var (
					k8sPlatform   *types.Platform
					k8sAgent      *common.SMExpect
					brokerID      string
					planCatalogID string
				)

				assertBrokerForPlatformByID := func(agent *common.SMExpect, brokerID interface{}, status int) {
					k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceBrokersURL, brokerID.(string))).
						Expect().
						Status(status)
				}

				assertBrokersForPlatformWithQuery := func(agent *common.SMExpect, query map[string]interface{}, brokers ...interface{}) {
					q := url.Values{}
					for k, v := range query {
						q.Set(k, fmt.Sprint(v))
					}
					queryString := q.Encode()
					result := agent.ListWithQuery(web.ServiceBrokersURL, queryString).Path("$[*].id").Array()
					result.Length().Equal(len(brokers))
					if len(brokers) > 0 {
						result.ContainsOnly(brokers...)
					}
				}

				assertBrokersForPlatform := func(agent *common.SMExpect, brokers ...interface{}) {
					assertBrokersForPlatformWithQuery(agent, nil, brokers...)
				}

				BeforeEach(func() {
					k8sPlatformJSON := common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
					k8sPlatform = common.RegisterPlatformInSM(k8sPlatformJSON, ctx.SMWithOAuth, map[string]string{})
					k8sAgent = &common.SMExpect{Expect: ctx.SM.Builder(func(req *httpexpect.Request) {
						username, password := k8sPlatform.Credentials.Basic.Username, k8sPlatform.Credentials.Basic.Password
						req.WithBasicAuth(username, password)
					})}

					plan := common.GeneratePaidTestPlan()
					planCatalogID = gjson.Get(plan, "id").String()
					service := common.GenerateTestServiceWithPlans(plan)
					catalog := common.NewEmptySBCatalog()
					catalog.AddService(service)
					brokerID = ctx.RegisterBrokerWithCatalog(catalog).Broker.ID
				})

				AfterEach(func() {
					ctx.CleanupAdditionalResources()
				})

				Context("with no visibilities for any of the broker's plans", func() {
					It("should return not found", func() {
						assertBrokersForPlatform(k8sAgent, nil...)
						assertBrokerForPlatformByID(k8sAgent, brokerID, http.StatusNotFound)
					})

					It("should not list broker with field query broker id", func() {
						assertBrokersForPlatformWithQuery(k8sAgent,
							map[string]interface{}{
								"fieldQuery": fmt.Sprintf("broker_id eq '%s'", brokerID),
							}, nil...)
					})
				})

				Context("with public visibility for broker's plan", func() {
					BeforeEach(func() {
						ctx.TestContextData.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(planCatalogID, "", "")
					})
					It("should return one broker", func() {
						assertBrokersForPlatform(k8sAgent, brokerID)
						assertBrokerForPlatformByID(k8sAgent, brokerID, http.StatusOK)
					})
				})

				Context("with visibility for platform and one of the broker's plans", func() {
					BeforeEach(func() {
						ctx.TestContextData.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(planCatalogID, k8sPlatform.ID, "")
					})

					It("should return the broker", func() {
						assertBrokersForPlatform(k8sAgent, brokerID)
						assertBrokerForPlatformByID(k8sAgent, brokerID, http.StatusOK)
					})
				})
			})

			Describe("POST", func() {
				verifyPOSTWhenCatalogFieldHasValue := func(responseVerifier func(r *httpexpect.Response), fieldPath string, fieldValue interface{}) {
					BeforeEach(func() {
						catalog, err := sjson.Set(string(brokerServer.Catalog), fieldPath, fieldValue)
						Expect(err).ToNot(HaveOccurred())

						brokerServer.Catalog = SBCatalog(catalog)
					})

					It("returns correct response", func() {
						responseVerifier(ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).Expect())

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
					})
				}

				Context("when content type is not JSON", func() {
					It("returns 415", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithText("text").
							Expect().
							Status(http.StatusUnsupportedMediaType).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
					})
				})

				Context("when request body is not a valid JSON", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
							WithText("invalid json").
							WithHeader("content-type", "application/json").
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
					})
				})

				Context("when request body contains protected labels", func() {
					BeforeEach(func() {
						postBrokerRequestWithLabels.SetLabels(Object{
							TenantLabelKey: []string{"test-tenant"},
						})
					})

					It("returns 400", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithLabels).
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
					})

					Context("when request body contains multiple label objects", func() {
						It("returns 400", func() {
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
								WithHeader("Content-Type", "application/json").
								WithBytes([]byte(fmt.Sprintf(`{
                        "name":        "broker-with-labels",
                        "broker_url":  "%s",
                        "description": "desc",
                        "credentials": {
                           "basic": {
                              "username": "%s",
                              "password": "%s"
                           }
                        },
                        "labels": {},
                        "labels": {
                           "%s":["test-tenant"]
                        }  
                     }`, brokerWithLabelsServer.URL(), brokerWithLabelsServer.Username, brokerWithLabelsServer.Password, TenantLabelKey))).
								Expect().
								Status(http.StatusBadRequest).
								JSON().Object().Value("description").String().Contains("invalid json: duplicate key labels")

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
						})
					})
				})

				Context("when a request body field is missing", func() {
					assertPOSTReturns400WhenFieldIsMissing := func(field string) {
						BeforeEach(func() {
							delete(postBrokerRequestWithNoLabels, field)
							delete(expectedBrokerResponse, field)
						})

						It("returns 400", func() {
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
								Expect().
								Status(http.StatusBadRequest).
								JSON().Object().
								Keys().Contains("error", "description")

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
						})
					}

					assertPOSTReturns201WhenFieldIsMissing := func(field string) {
						BeforeEach(func() {
							delete(postBrokerRequestWithNoLabels, field)
							delete(expectedBrokerResponse, field)
						})

						It("returns 201", func() {
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
								Expect().
								Status(http.StatusCreated).
								JSON().Object().
								ContainsMap(expectedBrokerResponse).
								Keys().NotContains("services").Contains("credentials")

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})

						Specify("the whole catalog is returned from the repository in the brokers catalog field", func() {
							id := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
								Expect().
								Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

							byID := query.ByField(query.EqualsOperator, "id", id)
							brokerFromDB, err := repository.Get(context.TODO(), types.ServiceBrokerType, byID)
							Expect(err).ToNot(HaveOccurred())

							Expect(string(brokerFromDB.(*types.ServiceBroker).Catalog)).To(MatchJSON(string(brokerServer.Catalog)))
						})
					}

					Context("when name field is missing", func() {
						assertPOSTReturns400WhenFieldIsMissing("name")
					})

					Context("when broker_url field is missing", func() {
						assertPOSTReturns400WhenFieldIsMissing("broker_url")
					})

					Context("when credentials field is missing", func() {
						assertPOSTReturns400WhenFieldIsMissing("credentials")
					})

					Context("when description field is missing", func() {
						assertPOSTReturns201WhenFieldIsMissing("description")
					})
				})

				Context("when request body has invalid field", func() {
					Context("when name field is too long", func() {
						BeforeEach(func() {
							length := 500
							brokerName := make([]rune, length)
							for i := range brokerName {
								brokerName[i] = 'a'
							}
							postBrokerRequestWithLabels["name"] = string(brokerName)
						})

						It("returns 400", func() {
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithLabels).
								Expect().
								Status(http.StatusBadRequest).
								JSON().Object().
								Keys().Contains("error", "description")

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
						})
					})
				})

				Context("when obtaining the broker catalog fails because the broker is not reachable", func() {
					BeforeEach(func() {
						postBrokerRequestWithNoLabels["broker_url"] = "http://localhost:12345"
					})

					It("returns 502", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
							Expect().
							Status(http.StatusBadGateway).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("when the broker returns 404 for catalog", func() {
					BeforeEach(func() {
						brokerServer.CatalogHandler = func(rw http.ResponseWriter, req *http.Request) {
							SetResponse(rw, http.StatusNotFound, Object{})
						}
					})

					It("returns 400", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
							Expect().
							Status(http.StatusBadRequest)
					})
				})

				Context("when the broker call for catalog times out", func() {
					const (
						timeoutDuration             = time.Millisecond * 500
						additionalDelayAfterTimeout = time.Second
					)

					BeforeEach(func() {
						settings := ctx.Config.HTTPClient
						settings.ResponseHeaderTimeout = timeoutDuration
						httpclient.SetHTTPClientGlobalSettings(settings)
						httpclient.Configure()
						brokerServer.CatalogHandler = func(rw http.ResponseWriter, req *http.Request) {
							catalogStopDuration := timeoutDuration + additionalDelayAfterTimeout
							continueCtx, _ := context.WithTimeout(req.Context(), catalogStopDuration)

							<-continueCtx.Done()

							SetResponse(rw, http.StatusTeapot, Object{})
						}
					})

					AfterEach(func() {
						httpclient.SetHTTPClientGlobalSettings(ctx.Config.HTTPClient)
						httpclient.Configure()
					})

					It("returns 502", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
							Expect().
							Status(http.StatusBadGateway).JSON().Object().Value("description").String().Contains("could not reach service broker")
					})
				})

				Context("tls", func() {
					var settings *httpclient.Settings
					BeforeEach(func() {
						settings = ctx.Config.HTTPClient
					})
					Context("mutual tls", func() {
						JustBeforeEach(func() {
							settings.SkipSSLValidation = false
							settings.RootCACertificates = []string{tls_settings.BrokerRootCertificate}
							httpclient.SetHTTPClientGlobalSettings(settings)
							httpclient.Configure()
							http.DefaultTransport.(*http.Transport).TLSClientConfig.ServerName = "localhost"

						})
						JustAfterEach(func() {
							http.DefaultTransport.(*http.Transport).TLSClientConfig = nil
							settings.TLSCertificates = []tls.Certificate{}
							settings.ServerCertificate = ""
							settings.ServerCertificateKey = ""
							settings.SkipSSLValidation = true
							settings.RootCACertificates = []string{}
							httpclient.SetHTTPClientGlobalSettings(settings)
							httpclient.Configure()
						})

						Context("server manager certificate is valid", func() {
							BeforeEach(func() {
								settings.ServerCertificate = tls_settings.ServerManagerCertificate
								settings.ServerCertificateKey = tls_settings.ServerManagerCertificateKey

							})
							When("sm provided credentials requested", func() {
								It("ok", func() {
									res := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerBasicServerMtls).
										Expect().
										Status(http.StatusCreated).JSON().Object()
									expectedResponse := Object{
										"name":        postBrokerBasicServerMtls["name"],
										"broker_url":  brokerServerWithSMCertficate.URL(),
										"description": postBrokerBasicServerMtls["description"],
										"credentials": Object{
											"sm_provided_tls_credentials": true,
										},
									}
									res.ContainsMap(expectedResponse)
									res.NotContainsKey("tls")
									assertInvocationCount(brokerServerWithSMCertficate.CatalogEndpointRequests, 1)
								})
							})
							When("mtls with basic", func() {
								It("should  succeed", func() {
									req := CopyObject(postBrokerBasicServerMtls)
									req["credentials"].(Object)["basic"] = Object{
										"username": brokerServerWithSMCertficate.Username,
										"password": brokerServerWithSMCertficate.Password,
									}
									res := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(req).
										Expect().Status(http.StatusCreated).JSON().Object()
									res.ContainsMap(expectedBrokerServerWithServiceManagerMtlsAndBasicAuth)
									assertInvocationCount(brokerServerWithSMCertficate.CatalogEndpointRequests, 1)
								})
							})

							When("no credentials are provided", func() {
								It("should fail", func() {
									req := CopyObject(postBrokerBasicServerMtls)
									delete(req, "credentials")
									ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(req).
										Expect().
										Status(http.StatusBadRequest)

								})

							})
							Context("broker with its own tls configuration", func() {
								It("sends request to a broker that supports only service manager certificate, should fail", func() {
									//check if only one certificate is sent, otherwise the connection would'nt fail
									ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithTLSToWrongBrokerServer).
										Expect().
										Status(http.StatusBadGateway)
								})
							})
							Context("broker client certificate and basic auth are configured", func() {
								It("should succeed", func() {
									reply := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithTLSandBasic).
										Expect().
										Status(http.StatusCreated).
										JSON().Object()
									reply.ContainsMap(expectedBrokerResponseTLS)
									assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 1)
								})

							})

						})

						Context("server manager certificate is expired", func() {
							BeforeEach(func() {
								settings.ServerCertificate = tls_settings.ExpiredServerManagerCertficate
								settings.ServerCertificateKey = tls_settings.ExpiredServerManagerCertificateKey
							})

							It("returns error", func() {
								ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerBasicServerMtls).
									Expect().
									Status(http.StatusBadGateway).Body()

							})

							When("broker certificate is configured", func() {
								It("should succeed", func() {
									reply := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithTLSandBasic).
										Expect().
										Status(http.StatusCreated).
										JSON().Object()
									reply.ContainsMap(expectedBrokerResponseTLS)
									assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 1)
								})
							})

						})

					})

					Context("broker client certificate", func() {
						BeforeEach(func() {
							settings := ctx.Config.HTTPClient
							settings.SkipSSLValidation = true
							httpclient.SetHTTPClientGlobalSettings(settings)
							httpclient.Configure()
						})

						Context("when broker basic and user auth are both configured", func() {
							It("returns StatusCreated", func() {
								reply := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithTLSandBasic).
									Expect().
									Status(http.StatusCreated).
									JSON().Object()
								reply.ContainsMap(expectedBrokerResponseTLS)
								assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 1)
							})
						})

						Context("when broker is behind tls but not valid certs are configured", func() {
							It("returns StatusBadGateway", func() {
								ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithTLSNoCert).
									Expect().
									Status(http.StatusBadGateway).
									JSON().Object()
								assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 0)
							})
						})

						//actually the broker return invalid credentials, however the response is converted by the catalog fetch into badRequest
						Context("when broker tls settings are valid but basic auth credentials are missing", func() {
							It("returns StatusCreated", func() {
								ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithTLS).
									Expect().
									Status(http.StatusCreated)
								assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 1)
							})
						})
					})

				})

				Context("when the broker catalog is incomplete", func() {
					verifyPOSTWhenCatalogFieldIsMissing := func(responseVerifier func(r *httpexpect.Response), fieldPath string) {
						BeforeEach(func() {
							catalog, err := sjson.Delete(string(NewRandomSBCatalog()), fieldPath)
							Expect(err).ToNot(HaveOccurred())

							brokerServer.Catalog = SBCatalog(catalog)
						})

						It("returns correct response", func() {
							responseVerifier(ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).Expect())

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

						})
					}

					Context("when the broker catalog contains an incomplete service", func() {
						Context("that has an empty catalog id", func() {
							verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, "services.0.id")
						})

						Context("that has an empty catalog name", func() {
							verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, "services.0.name")
						})

						Context("that has an empty description", func() {
							verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusCreated).JSON().Object().Keys().NotContains("services").Contains("credentials")
							}, "services.0.description")
						})

						Context("that has an empty description", func() {
							verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.plans.0.description")
						})

						Context("that has invalid tags", func() {
							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.tags", "{invalid")
						})

						Context("that has invalid requires", func() {
							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.requires", "{invalid")
						})

						Context("that has invalid metadata", func() {
							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.metadata", "{invalid")
						})
					})

					Context("when broker catalog contains an invalid plan", func() {
						Context("that has an empty catalog id", func() {
							verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, "services.0.plans.0.id")
						})

						Context("that has an empty catalog name", func() {
							verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, "services.0.plans.0.name")
						})

						Context("that has an empty description", func() {
							verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, "services.0.plans.0.description")
						})

						Context("that has invalid metadata", func() {
							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.plans.0.metadata", "{invalid")
						})

						Context("that has invalid schemas", func() {
							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.plans.0.schemas", "{invalid")
						})

						Context("that has both supportedPlatforms and supportedPlatformNames", func() {

							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.plans.0.metadata", common.Object{"supportedPlatforms": []string{"a"}, "supportedPlatformNames": []string{"a"}})
						})

						Context("that has both supportedPlatforms and excludedPlatformNames", func() {

							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.plans.0.metadata", common.Object{"supportedPlatforms": []string{"a"}, "excludedPlatformNames": []string{"a"}})
						})

						Context("that has both supportedPlatformNames and excludedPlatformNames", func() {

							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.plans.0.metadata", common.Object{"supportedPlatformNames": []string{"a"}, "excludedPlatformNames": []string{"a"}})
						})

						Context(fmt.Sprintf("that has same name with the reserved reference plan: %s", instance_sharing.ReferencePlanName), func() {
							verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
							}, "services.0.plans.0.name", instance_sharing.ReferencePlanName)
						})
					})
				})

				Context("when fetching catalog fails", func() {
					BeforeEach(func() {
						brokerServer.CatalogHandler = func(w http.ResponseWriter, req *http.Request) {
							SetResponse(w, http.StatusInternalServerError, Object{})
						}
					})

					It("returns 400", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
							WithJSON(postBrokerRequestWithNoLabels).
							Expect().Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
					})
				})

				Context("when fetching the catalog is successful", func() {
					assertPOSTReturns201 := func() {
						It("returns 201", func() {
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
								Expect().
								Status(http.StatusCreated).
								JSON().Object().
								ContainsMap(expectedBrokerResponse).
								Keys().NotContains("services").Contains("credentials")

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})
					}

					Context("when broker URL does not end with trailing slash", func() {
						BeforeEach(func() {
							postBrokerRequestWithNoLabels["broker_url"] = strings.TrimRight(cast.ToString(postBrokerRequestWithNoLabels["broker_url"]), "/")
							expectedBrokerResponse["broker_url"] = strings.TrimRight(cast.ToString(expectedBrokerResponse["broker_url"]), "/")
						})

						assertPOSTReturns201()
					})

					Context("when broker URL ends with trailing slash", func() {
						BeforeEach(func() {
							postBrokerRequestWithNoLabels["broker_url"] = cast.ToString(postBrokerRequestWithNoLabels["broker_url"]) + "/"
							expectedBrokerResponse["broker_url"] = cast.ToString(expectedBrokerResponse["broker_url"]) + "/"
						})

						assertPOSTReturns201()
					})
				})

				Context("when broker with name already exists", func() {
					It("returns 409", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
							Expect().
							Status(http.StatusCreated)

						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
							Expect().
							Status(http.StatusConflict).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 2)
					})
				})

				Context("Labelled", func() {
					Context("When labels are valid", func() {
						It("should return 201", func() {
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
								WithJSON(postBrokerRequestWithLabels).
								Expect().Status(http.StatusCreated).JSON().Object().Keys().Contains("id", "labels")
						})

						When("When creating labeled broker containing a separator not as a standalone word", func() {
							It("should return 201", func() {
								labels[fmt.Sprintf("containing_%s_separator", query.Separator)] = Array{"val"}
								ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
									WithJSON(postBrokerRequestWithLabels).
									Expect().Status(http.StatusCreated).JSON().Object().Keys().Contains("id", "labels")
							})
						})
					})

					Context("When creating labeled broker with a forbidden separator", func() {
						It("Should return 400 if the separator is a standalone word", func() {
							labels[fmt.Sprintf("containing %s separator", query.Separator)] = Array{"val"}
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
								WithJSON(postBrokerRequestWithLabels).
								Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").String().Contains("cannot contain whitespaces")
						})
					})

					Context("When label key has new line", func() {
						It("Should return 400", func() {
							labels[`key with
   new line`] = Array{"label-value"}
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
								WithJSON(postBrokerRequestWithLabels).
								Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").String().Contains("cannot contain whitespaces")
						})
					})

					Context("When label value has new line", func() {
						It("Should return 400", func() {
							labels["cluster_id"] = Array{`{
   "key": "k1",
   "val": "val1"
   }`}
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
								WithJSON(postBrokerRequestWithLabels).
								Expect().Status(http.StatusBadRequest)
						})
					})
				})

				Context("Supported platforms", func() {
					Context("When a plan has supportedPlatforms in metadata", func() {
						verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
							r.Status(http.StatusCreated)
						}, "services.0.plans.0.metadata", common.Object{"supportedPlatforms": []string{"a"}})
					})

					Context("When a plan has supportedPlatformNames in metadata", func() {
						verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
							r.Status(http.StatusCreated)
						}, "services.0.plans.0.metadata", common.Object{"supportedPlatformNames": []string{"a"}})
					})
				})

				Context("transitive resources", func() {
					It("should be saved in the operation", func() {
						broker := t.ResourceBlueprint(ctx, ctx.SMWithOAuth, true)
						brokerID := broker["id"].(string)

						offerings, err := ctx.SMRepository.List(context.Background(), types.ServiceOfferingType, query.ByField(query.EqualsOperator, "broker_id", brokerID))
						Expect(err).ShouldNot(HaveOccurred())

						offeringIDs := make([]string, 0, offerings.Len())
						for i := 0; i < offerings.Len(); i++ {
							offeringIDs = append(offeringIDs, offerings.ItemAt(i).GetID())
						}
						plans, err := ctx.SMRepository.List(context.Background(), types.ServicePlanType, query.ByField(query.InOperator, "service_offering_id", offeringIDs...))
						Expect(err).ShouldNot(HaveOccurred())

						planIDs := make([]string, 0, plans.Len())
						for i := 0; i < plans.Len(); i++ {
							planIDs = append(planIDs, plans.ItemAt(i).GetID())
						}
						visibilities, err := ctx.SMRepository.List(context.Background(), types.VisibilityType, query.ByField(query.InOperator, "service_plan_id", planIDs...))
						Expect(err).ShouldNot(HaveOccurred())

						transitiveResourcesExpectedCount := offerings.Len() + plans.Len() + visibilities.Len()
						operation, err := ctx.SMRepository.Get(context.Background(), types.OperationType,
							query.ByField(query.EqualsOperator, "resource_id", brokerID),
							query.ByField(query.EqualsOperator, "type", string(types.CREATE)),
							query.OrderResultBy("paging_sequence", query.DescOrder))
						Expect(err).ShouldNot(HaveOccurred())

						transitiveResources := operation.(*types.Operation).TransitiveResources

						transitiveResourcesActualCount := 0
						for _, tr := range transitiveResources {
							// Do not count the notifications and resources which are not for created
							if tr.Type != types.NotificationType && tr.OperationType == types.CREATE {
								transitiveResourcesActualCount++
							}
						}
						Expect(transitiveResourcesActualCount).To(Equal(transitiveResourcesExpectedCount))
					})
				})

				Context("when broker is responding slow", func() {
					It("should timeout", func() {
						brokerServer.CatalogHandler = func(rw http.ResponseWriter, req *http.Request) {
							rw.WriteHeader(http.StatusOK)
							if fl, ok := rw.(http.Flusher); ok {
								for i := 0; i < 50; i++ {
									fmt.Fprintf(rw, "Chunk %d", i)
									fl.Flush()
									time.Sleep(time.Millisecond * 100)
								}
							}
						}

						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
							Expect().
							Status(http.StatusGatewayTimeout)
					})
				})
			})

			Describe("PATCH", func() {
				var brokerID, brokerIDWithTLS string

				assertRepositoryReturnsExpectedCatalogAfterPatching := func(brokerID, expectedCatalog string) {
					ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
						WithJSON(Object{}).
						Expect()

					byID := query.ByField(query.EqualsOperator, "id", brokerID)
					brokerFromDB, err := repository.Get(context.TODO(), types.ServiceBrokerType, byID)
					Expect(err).ToNot(HaveOccurred())

					Expect(string(brokerFromDB.(*types.ServiceBroker).Catalog)).To(MatchJSON(expectedCatalog))
				}

				BeforeEach(func() {
					reply := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
						Expect().
						Status(http.StatusCreated).
						JSON().Object().
						ContainsMap(expectedBrokerResponse)

					replyWithTLS := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithTLSandBasic).
						Expect().
						Status(http.StatusCreated).
						JSON().Object()

					brokerIDWithTLS = replyWithTLS.Value("id").String().Raw()
					brokerID = reply.Value("id").String().Raw()

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
					brokerServer.ResetCallHistory()
					brokerServerWithBrokerCertificate.ResetCallHistory()
				})

				Context("when content type is not JSON", func() {
					It("returns 415", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
							WithText("text").
							Expect().Status(http.StatusUnsupportedMediaType).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
					})
				})

				Context("when broker is missing", func() {
					It("returns 404", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/no_such_id").
							WithJSON(postBrokerRequestWithNoLabels).
							Expect().Status(http.StatusNotFound).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				Context("when request body is not valid JSON", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
							WithText("invalid json").
							WithHeader("content-type", "application/json").
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				Context("when request body contains invalid credentials", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
							WithJSON(Object{"credentials": "123"}).
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				Context("when request body contains incomplete credentials", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
							WithJSON(Object{"credentials": Object{"basic": Object{"password": ""}}}).
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				Context("when broker with the name already exists", func() {
					var anotherTestBroker Object
					var anotherBrokerServer *BrokerServer

					BeforeEach(func() {
						anotherBrokerServer = NewBrokerServer()
						anotherBrokerServer.Username = "username"
						anotherBrokerServer.Password = "password"
						anotherTestBroker = Object{
							"name":        "another_name",
							"broker_url":  anotherBrokerServer.URL(),
							"description": "another_description",
							"credentials": Object{
								"basic": Object{
									"username": anotherBrokerServer.Username,
									"password": anotherBrokerServer.Password,
								},
							},
						}
					})

					AfterEach(func() {
						if anotherBrokerServer != nil {
							anotherBrokerServer.Close()
						}
					})

					It("returns 409", func() {
						ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
							WithJSON(anotherTestBroker).
							Expect().
							Status(http.StatusCreated)

						assertInvocationCount(anotherBrokerServer.CatalogEndpointRequests, 1)

						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
							WithJSON(anotherTestBroker).
							Expect().Status(http.StatusConflict).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
					})
				})

				Context("when credentials are updated", func() {

					Context("when broker is behind tls", func() {

						It("when credentials contain an invalid certificate", func() {
							updatedCredentials := Object{
								"credentials": Object{
									"tls": Object{
										"client_certificate": tls_settings.InvalidClientCertificate,
										"client_key":         tls_settings.InvalidClientKey,
									},
								},
							}
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithTLS).WithJSON(updatedCredentials).
								Expect().
								Status(http.StatusBadGateway).Body().Contains("could not reach service broker")
							assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 0)
						})

						It("when credentials contain valid certificate", func() {
							updatedCredentials := Object{
								"credentials": Object{
									"tls": Object{
										"client_certificate": tls_settings.ClientCertificate,
										"client_key":         tls_settings.ClientKey,
									},
								},
							}
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithTLS).WithJSON(updatedCredentials).
								Expect().
								Status(http.StatusOK)
							assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 1)
						})

						It("when tls credentials and sm provided credential", func() {
							updatedCredentials := Object{
								"credentials": Object{
									"tls": Object{
										"client_certificate": tls_settings.ClientCertificate,
										"client_key":         tls_settings.ClientKey,
									},
									"sm_provided_tls_credentials": true,
								},
							}
							reply := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithTLS).WithJSON(updatedCredentials).
								Expect()
							reply.Status(http.StatusBadRequest)
							reply.Body().Contains("only one of the options could be set, SM provided credentials or tls")
							assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 0)
						})

						It("when emptying tls custom credentials but sm default tls is not available", func() {
							updatedCredentials := Object{
								"credentials": Object{
									"tls": Object{
										"client_certificate": "",
										"client_key":         "",
									},
									"sm_provided_tls_credentials": true,
								},
							}
							reply := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithTLS).WithJSON(updatedCredentials).
								Expect()
							reply.Status(http.StatusBadRequest)
							reply.Body().Contains("SM provided credentials are not supported in this region")
							assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 0)
						})

						It("when tls is used ", func() {
							updatedCredentials := Object{
								"credentials": Object{
									"tls": Object{
										"client_certificate": tls_settings.ClientCertificate,
										"client_key":         tls_settings.ClientKey,
									},
								},
							}
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithTLS).WithJSON(updatedCredentials).
								Expect().
								Status(http.StatusOK)
							assertInvocationCount(brokerServerWithBrokerCertificate.CatalogEndpointRequests, 1)
						})
						Context("default mTLS", func() {
							var settings *httpclient.Settings
							var brokerIDWithMTLS string
							BeforeEach(func() {
								settings = ctx.Config.HTTPClient
								settings.SkipSSLValidation = false
								settings.RootCACertificates = []string{tls_settings.BrokerRootCertificate}
								settings.ServerCertificate = tls_settings.ServerManagerCertificate
								settings.ServerCertificateKey = tls_settings.ServerManagerCertificateKey
								httpclient.SetHTTPClientGlobalSettings(settings)
								httpclient.Configure()
								http.DefaultTransport.(*http.Transport).TLSClientConfig.ServerName = "localhost"

								replyWithMTLS := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerBasicServerMtls).
									Expect().
									Status(http.StatusCreated).
									JSON().Object()

								brokerIDWithMTLS = replyWithMTLS.Value("id").String().Raw()
								assertInvocationCount(brokerServerWithSMCertficate.CatalogEndpointRequests, 1)
								brokerServer.ResetCallHistory()
								brokerServerWithSMCertficate.ResetCallHistory()

							})
							AfterEach(func() {
								http.DefaultTransport.(*http.Transport).TLSClientConfig = nil
								settings.TLSCertificates = []tls.Certificate{}
								settings.ServerCertificate = ""
								settings.ServerCertificateKey = ""
								settings.SkipSSLValidation = true
								settings.RootCACertificates = []string{}
								httpclient.SetHTTPClientGlobalSettings(settings)
								httpclient.Configure()
							})

							Context("empty  basic credentials, provided credentials still exist", func() {
								It("should fail", func() {
									updatedBrokerJSON := Object{
										"name":        "updated_name",
										"description": "updated_description",
										"credentials": Object{
											"basic": Object{
												"username": "",
												"password": "",
											},
										},
									}
									reply := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithMTLS).WithJSON(updatedBrokerJSON).
										Expect()
									reply.Status(http.StatusOK)
									assertInvocationCount(brokerServerWithSMCertficate.CatalogEndpointRequests, 1)

								})
							})
							Context("empty all credentials", func() {
								It("should fail", func() {
									updatedBrokerJSON := Object{
										"name":        "updated_name",
										"description": "updated_description",
										"credentials": Object{
											"sm_provided_tls_credentials": false,
											"basic": Object{
												"username": "",
												"password": "",
											},
										},
									}
									reply := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithMTLS).WithJSON(updatedBrokerJSON).
										Expect()
									reply.Status(http.StatusBadRequest)
									reply.Body().Contains("missing broker credentials: SM provided mtls , basic or tls credentials are required")
									assertInvocationCount(brokerServerWithSMCertficate.CatalogEndpointRequests, 0)

								})
							})
							Context("use sm default credentials", func() {
								It("should succeed", func() {
									updatedBrokerJSON := Object{
										"name":        "updated_name",
										"description": "updated_description",
										"credentials": Object{
											"basic": Object{
												"username": "",
												"password": "",
											},
											"sm_provided_tls_credentials": true,
										},
									}

									reply := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerIDWithMTLS).WithJSON(updatedBrokerJSON).
										Expect()
									reply.Status(http.StatusOK)
									assertInvocationCount(brokerServerWithSMCertficate.CatalogEndpointRequests, 1)
									byID := query.ByField(query.EqualsOperator, "id", brokerIDWithMTLS)
									dbObj, err := repository.Get(context.TODO(), types.ServiceBrokerType, byID)
									Expect(err).ToNot(HaveOccurred())
									brokerFromDB := dbObj.(*types.ServiceBroker)
									Expect(brokerFromDB.Credentials.Basic.Username).To(BeEmpty())

								})

							})
						})
					})

					It("only username is updated, should fail", func() {
						brokerServer.Username = "updatedUsername"
						brokerServer.Password = "updatedPassword"
						updatedCredentials := Object{
							"credentials": Object{
								"basic": Object{
									"username": brokerServer.Username,
								},
							},
						}
						reply := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
							WithJSON(updatedCredentials).
							Expect()
						reply.Status(http.StatusBadRequest)
						reply.Body().Contains("401 Unauthorized")
						assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)

					})

					It("valid basic auth returns 200", func() {
						brokerServer.Username = "updatedUsername"
						brokerServer.Password = "updatedPassword"
						updatedCredentials := Object{
							"credentials": Object{
								"basic": Object{
									"username": brokerServer.Username,
									"password": brokerServer.Password,
								},
							},
						}
						reply := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
							WithJSON(updatedCredentials).
							Expect().
							Status(http.StatusOK).
							JSON().Object()

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

						reply = ctx.SMWithOAuth.GET(web.ServiceBrokersURL + "/" + brokerID).
							Expect().
							Status(http.StatusOK).
							JSON().Object()
						reply.ContainsMap(expectedBrokerResponse)
					})
				})

				Context("when created_at provided in body", func() {
					It("should not change created_at", func() {
						createdAt := "2015-01-01T00:00:00Z"

						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
							WithJSON(Object{"created_at": createdAt}).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							ValueNotEqual("created_at", createdAt)

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

						ctx.SMWithOAuth.GET(web.ServiceBrokersURL+"/"+brokerID).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							ValueNotEqual("created_at", createdAt)
					})
				})

				Context("when new broker server is available", func() {
					var (
						updatedBrokerServer           *BrokerServer
						updatedBrokerJSON             Object
						expectedUpdatedBrokerResponse Object
					)

					BeforeEach(func() {
						updatedBrokerServer = NewBrokerServer()
						updatedBrokerServer.Username = "updated_user"
						updatedBrokerServer.Password = "updated_password"
						updatedBrokerJSON = Object{
							"name":        "updated_name",
							"description": "updated_description",
							"broker_url":  updatedBrokerServer.URL(),
							"credentials": Object{
								"basic": Object{
									"username": updatedBrokerServer.Username,
									"password": updatedBrokerServer.Password,
								},
							},
						}

						expectedUpdatedBrokerResponse = Object{
							"name":        updatedBrokerJSON["name"],
							"description": updatedBrokerJSON["description"],
							"broker_url":  updatedBrokerJSON["broker_url"],
						}
					})

					AfterEach(func() {
						if updatedBrokerServer != nil {
							updatedBrokerServer.Close()
						}
					})

					Context("when all updatable fields are updated at once", func() {
						It("returns 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
								WithJSON(updatedBrokerJSON).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsMap(expectedUpdatedBrokerResponse).
								Keys().NotContains("services", "credentials")

							assertInvocationCount(updatedBrokerServer.CatalogEndpointRequests, 1)

							ctx.SMWithOAuth.GET(web.ServiceBrokersURL+"/"+brokerID).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsMap(expectedUpdatedBrokerResponse).
								Keys().NotContains("services", "credentials")
						})
					})

					Context("when broker_url is changed and the credentials are correct", func() {
						It("returns 200 with basic", func() {
							updatedBrokerJSON := Object{
								"broker_url": updatedBrokerServer.URL(),
								"credentials": Object{
									"basic": Object{
										"username": brokerServer.Username,
										"password": brokerServer.Password,
									},
								},
							}
							updatedBrokerServer.Username = brokerServer.Username
							updatedBrokerServer.Password = brokerServer.Password

							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
								WithJSON(updatedBrokerJSON).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsKey("broker_url").
								Keys().NotContains("services", "credentials")

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
							assertInvocationCount(updatedBrokerServer.CatalogEndpointRequests, 1)

							ctx.SMWithOAuth.GET(web.ServiceBrokersURL+"/"+brokerID).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsKey("broker_url").
								Keys().NotContains("services", "credentials")
						})

						It("returns 200 with tls", func() {
							updatedBrokerJSON := Object{
								"broker_url": updatedBrokerServer.URL(),
								"credentials": Object{
									"tls": Object{
										"client_certificate": tls_settings.ClientCertificate,
										"client_key":         tls_settings.ClientKey,
									},
								},
							}

							updatedBrokerServer.Username = brokerServerWithBrokerCertificate.Username
							updatedBrokerServer.Password = brokerServerWithBrokerCertificate.Password

							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerIDWithTLS).
								WithJSON(updatedBrokerJSON).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsKey("broker_url").
								Keys().NotContains("services", "credentials")

							assertInvocationCount(updatedBrokerServer.CatalogEndpointRequests, 1)
						})
					})

					Context("when broker_url is changed but the credentials are wrong", func() {
						It("returns 400", func() {
							updatedBrokerJSON := Object{
								"broker_url": updatedBrokerServer.URL(),
							}
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
								WithJSON(updatedBrokerJSON).
								Expect().
								Status(http.StatusBadRequest).JSON().Object().Keys().
								Contains("error", "description")

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)

							ctx.SMWithOAuth.GET(web.ServiceBrokersURL+"/"+brokerID).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsMap(expectedBrokerResponse).
								Keys().NotContains("services", "credentials")
						})
					})

					Context("when broker_url is changed but the credentials are missing", func() {
						var updatedBrokerJSON Object

						BeforeEach(func() {
							updatedBrokerJSON = Object{
								"broker_url": updatedBrokerServer.URL(),
							}
						})

						Context("when broker is behind tls", func() {
							Context("credentials object is missing", func() {
								It("returns 400", func() {
									ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerIDWithTLS).
										WithJSON(updatedBrokerJSON).
										Expect().
										Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
								})
							})

							Context("client_certificate is missing", func() {
								BeforeEach(func() {
									updatedBrokerJSON["credentials"] = Object{
										"tls": Object{
											"client_key": "ck",
										},
									}
								})

								It("returns 400", func() {
									ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerIDWithTLS).
										WithJSON(updatedBrokerJSON).
										Expect().
										Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
								})
							})

							Context("client_key is missing", func() {
								BeforeEach(func() {
									updatedBrokerJSON["credentials"] = Object{
										"tls": Object{
											"client_certificate": "cc",
										},
									}
								})

								It("returns 400", func() {
									ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerIDWithTLS).
										WithJSON(updatedBrokerJSON).
										Expect().
										Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
								})
							})
						})

						Context("when broker is using basic credentials", func() {
							Context("credentials object is missing", func() {
								It("returns 400", func() {
									ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
										WithJSON(updatedBrokerJSON).
										Expect().
										Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
								})
							})

							Context("username is missing", func() {
								BeforeEach(func() {
									updatedBrokerJSON["credentials"] = Object{
										"basic": Object{
											"password": "b",
										},
									}
								})

								It("returns 400", func() {
									ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
										WithJSON(updatedBrokerJSON).
										Expect().
										Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
								})
							})

							Context("password is missing", func() {
								BeforeEach(func() {
									updatedBrokerJSON["credentials"] = Object{
										"basic": Object{
											"username": "a",
										},
									}
								})

								It("returns 400", func() {
									ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
										WithJSON(updatedBrokerJSON).
										Expect().
										Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
								})
							})
						})

					})
				})

				Context("when fields are updated one by one", func() {
					It("returns 200", func() {
						for _, prop := range []string{"name", "description"} {
							updatedBrokerJSON := Object{}
							updatedBrokerJSON[prop] = "updated"
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
								WithJSON(updatedBrokerJSON).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsMap(updatedBrokerJSON).
								Keys().NotContains("services", "credentials")

							ctx.SMWithOAuth.GET(web.ServiceBrokersURL+"/"+brokerID).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsMap(updatedBrokerJSON).
								Keys().NotContains("services", "credentials")

						}
						assertInvocationCount(brokerServer.CatalogEndpointRequests, 2)
					})
				})

				Context("when not updatable fields are provided in the request body", func() {
					Context("when broker id is provided in request body", func() {
						It("should not create the broker", func() {
							postBrokerRequestWithNoLabels = Object{"id": "123"}
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(postBrokerRequestWithNoLabels).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								NotContainsMap(postBrokerRequestWithNoLabels)

							ctx.SMWithOAuth.GET(web.ServiceBrokersURL + "/123").
								Expect().
								Status(http.StatusNotFound)

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})
					})

					Context("when unmodifiable fields are provided in the request body", func() {
						BeforeEach(func() {
							postBrokerRequestWithNoLabels = Object{
								"created_at": "2016-06-08T16:41:26Z",
								"updated_at": "2016-06-08T16:41:26Z",
								"services":   Array{Object{"name": "serviceName"}},
							}
						})

						It("should not change them", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(postBrokerRequestWithNoLabels).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								NotContainsMap(postBrokerRequestWithNoLabels)

							ctx.SMWithOAuth.List(web.ServiceBrokersURL).First().Object().
								ContainsMap(expectedBrokerResponse)

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})
					})
				})

				Context("when obtaining the broker catalog fails because the broker is not reachable", func() {
					BeforeEach(func() {
						postBrokerRequestWithNoLabels["broker_url"] = "http://localhost:12345"
					})

					It("returns 502", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).WithJSON(postBrokerRequestWithNoLabels).
							Expect().
							Status(http.StatusBadGateway).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("when fetching the broker catalog fails", func() {
					BeforeEach(func() {
						brokerServer.CatalogHandler = func(w http.ResponseWriter, req *http.Request) {
							SetResponse(w, http.StatusInternalServerError, Object{})
						}
					})

					It("returns an error", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).
							WithJSON(postBrokerRequestWithNoLabels).
							Expect().Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")

						assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
					})
				})

				Context("when the broker catalog is modified", func() {
					Context("when a new service offering with a plan existing for another service offering is added", func() {
						var anotherServiceID string
						var existingPlanID string

						BeforeEach(func() {
							existingServicePlan := gjson.Get(string(brokerServer.Catalog), "services.0.plans.0").String()
							existingPlanID = gjson.Get(existingServicePlan, "id").String()
							anotherServiceWithSamePlan, err := sjson.Set(GenerateTestServiceWithPlans(), "plans.-1", JSONToMap(existingServicePlan))
							Expect(err).ShouldNot(HaveOccurred())

							anotherService := JSONToMap(anotherServiceWithSamePlan)
							anotherServiceID = anotherService["id"].(string)
							Expect(anotherServiceID).ToNot(BeEmpty())

							catalog, err := sjson.Set(string(brokerServer.Catalog), "services.-1", anotherService)
							Expect(err).ShouldNot(HaveOccurred())

							brokerServer.Catalog = SBCatalog(catalog)
						})

						It("is returned from the Services API associated with the correct broker", func() {
							ctx.SMWithOAuth.List(web.ServiceOfferingsURL).
								Path("$[*].catalog_id").Array().NotContains(anotherServiceID)
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(Object{}).
								Expect().
								Status(http.StatusOK)

							By("updating broker again with 2 services with identical plans, should succeed")
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(Object{}).
								Expect().
								Status(http.StatusOK)

							servicesJsonResp := ctx.SMWithOAuth.List(web.ServiceOfferingsURL)
							servicesJsonResp.Path("$[*].catalog_id").Array().Contains(anotherServiceID)
							servicesJsonResp.Path("$[*].broker_id").Array().Contains(brokerID)

							var soID string
							for _, so := range servicesJsonResp.Iter() {
								sbID := so.Object().Value("broker_id").String().Raw()
								Expect(sbID).ToNot(BeEmpty())

								catalogID := so.Object().Value("catalog_id").String().Raw()
								Expect(catalogID).ToNot(BeEmpty())

								if catalogID == anotherServiceID && sbID == brokerID {
									soID = so.Object().Value("id").String().Raw()
									Expect(soID).ToNot(BeEmpty())
									break
								}
							}

							plansJsonResp := ctx.SMWithOAuth.List(web.ServicePlansURL)
							plansJsonResp.Path("$[*].catalog_id").Array().Contains(existingPlanID)
							plansJsonResp.Path("$[*].service_offering_id").Array().Contains(soID)

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 2)
						})

						It("is returned from the repository as part of the brokers catalog field", func() {
							assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, string(brokerServer.Catalog))
						})
					})

					Context("when a new service offering with new plans is added", func() {
						var anotherServiceID string
						var anotherPlanID string

						BeforeEach(func() {
							anotherPlan := JSONToMap(GeneratePaidTestPlan())
							anotherPlanID = anotherPlan["id"].(string)
							anotherServiceWithAnotherPlan, err := sjson.Set(GenerateTestServiceWithPlans(), "plans.-1", anotherPlan)
							Expect(err).ShouldNot(HaveOccurred())

							anotherService := JSONToMap(anotherServiceWithAnotherPlan)
							anotherServiceID = anotherService["id"].(string)
							Expect(anotherServiceID).ToNot(BeEmpty())

							catalog, err := sjson.Set(string(brokerServer.Catalog), "services.-1", anotherService)
							Expect(err).ShouldNot(HaveOccurred())

							brokerServer.Catalog = SBCatalog(catalog)
						})

						It("is returned from the Services API associated with the correct broker", func() {
							ctx.SMWithOAuth.List(web.ServiceOfferingsURL).
								Path("$[*].catalog_id").Array().NotContains(anotherServiceID)
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(Object{}).
								Expect().
								Status(http.StatusOK)
							servicesJsonResp := ctx.SMWithOAuth.List(web.ServiceOfferingsURL)
							servicesJsonResp.Path("$[*].catalog_id").Array().Contains(anotherServiceID)
							servicesJsonResp.Path("$[*].broker_id").Array().Contains(brokerID)

							var soID string
							for _, so := range servicesJsonResp.Iter() {
								sbID := so.Object().Value("broker_id").String().Raw()
								Expect(sbID).ToNot(BeEmpty())

								catalogID := so.Object().Value("catalog_id").String().Raw()
								Expect(catalogID).ToNot(BeEmpty())

								if catalogID == anotherServiceID && sbID == brokerID {
									soID = so.Object().Value("id").String().Raw()
									Expect(soID).ToNot(BeEmpty())
									break
								}
							}

							plansJsonResp := ctx.SMWithOAuth.List(web.ServicePlansURL)
							plansJsonResp.Path("$[*].catalog_id").Array().Contains(anotherPlanID)
							plansJsonResp.Path("$[*].service_offering_id").Array().Contains(soID)

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})

						It("is returned from the repository as part of the brokers catalog field", func() {
							assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, string(brokerServer.Catalog))
						})

						It("is added to the broker update operation as transitive resource", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(Object{}).
								Expect().
								Status(http.StatusOK)

							operationObj, err := ctx.SMRepository.Get(context.Background(), types.OperationType,
								query.ByField(query.EqualsOperator, "resource_id", brokerID),
								query.ByField(query.EqualsOperator, "type", string(types.UPDATE)),
								query.OrderResultBy("paging_sequence", query.DescOrder))
							Expect(err).ShouldNot(HaveOccurred())
							operation := operationObj.(*types.Operation)

							common.AssertTransitiveResources(operation, TransitiveResourcesExpectation{
								CreatedOfferings:     1,
								CreatedPlans:         1,
								CreatedNotifications: 1,
							})

							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(Object{}).
								Expect().
								Status(http.StatusOK)

							operationObj, err = ctx.SMRepository.Get(context.Background(), types.OperationType,
								query.ByField(query.EqualsOperator, "resource_id", brokerID),
								query.ByField(query.EqualsOperator, "type", string(types.UPDATE)),
								query.OrderResultBy("paging_sequence", query.DescOrder))
							Expect(err).ShouldNot(HaveOccurred())
							operation = operationObj.(*types.Operation)

							common.AssertTransitiveResources(operation, TransitiveResourcesExpectation{
								CreatedOfferings:     0,
								CreatedPlans:         0,
								CreatedNotifications: 1,
							})
						})
					})

					verifyPATCHWhenCatalogFieldIsMissing := func(responseVerifier func(r *httpexpect.Response), shouldUpdateCatalog bool, fieldPath string) {
						var expectedCatalog string

						BeforeEach(func() {
							catalog, err := sjson.Delete(string(brokerServer.Catalog), fieldPath)
							Expect(err).ToNot(HaveOccurred())
							if !shouldUpdateCatalog {
								expectedCatalog = string(brokerServer.Catalog)
							} else {
								expectedCatalog = string(catalog)
							}
							brokerServer.Catalog = SBCatalog(catalog)
						})

						It("returns correct response", func() {
							responseVerifier(ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).WithJSON(Object{}).Expect())

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})

						Specify("the catalog is correctly returned by the repository", func() {
							assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, expectedCatalog)

						})
					}

					verifyPATCHWhenCatalogFieldHasValue := func(responseVerifier func(r *httpexpect.Response), shouldUpdateCatalog bool, fieldPath string, fieldValue interface{}) {
						var expectedCatalog string

						BeforeEach(func() {
							catalog, err := sjson.Set(string(brokerServer.Catalog), fieldPath, fieldValue)
							Expect(err).ToNot(HaveOccurred())
							if !shouldUpdateCatalog {
								expectedCatalog = string(brokerServer.Catalog)
							} else {
								expectedCatalog = string(catalog)
							}
							brokerServer.Catalog = SBCatalog(catalog)
						})

						It("returns correct response", func() {
							responseVerifier(ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).WithJSON(Object{}).Expect())

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})

						Specify("the catalog is correctly returned by the repository", func() {
							assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, expectedCatalog)
						})
					}

					Context("when a new service offering is added", func() {
						var anotherServiceID string

						BeforeEach(func() {
							anotherService := JSONToMap(GenerateTestServiceWithPlans())
							anotherServiceID = anotherService["id"].(string)
							Expect(anotherServiceID).ToNot(BeEmpty())

							currServices, err := sjson.Set(string(brokerServer.Catalog), "services.-1", anotherService)
							Expect(err).ShouldNot(HaveOccurred())

							brokerServer.Catalog = SBCatalog(currServices)
						})

						It("is returned from the Services API associated with the correct broker", func() {
							ctx.SMWithOAuth.List(web.ServiceOfferingsURL).
								Path("$[*].catalog_id").Array().NotContains(anotherServiceID)

							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(Object{}).
								Expect().
								Status(http.StatusOK)

							jsonResp := ctx.SMWithOAuth.List(web.ServiceOfferingsURL)
							jsonResp.Path("$[*].catalog_id").Array().Contains(anotherServiceID)
							jsonResp.Path("$[*].broker_id").Array().Contains(brokerID)

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})

						It("is returned from the repository as part of the brokers catalog field", func() {
							assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, string(brokerServer.Catalog))

						})
					})

					Context("when an existing service offering is removed", func() {
						var serviceOfferingID string
						var planIDsForService []string

						BeforeEach(func() {
							planIDsForService = make([]string, 0)

							catalogServiceID := gjson.Get(string(brokerServer.Catalog), "services.0.id").Str
							Expect(catalogServiceID).ToNot(BeEmpty())

							serviceOffering := ctx.SMWithOAuth.ListWithQuery(web.ServiceOfferingsURL,
								fmt.Sprintf("fieldQuery=broker_id eq '%s' and catalog_id eq '%s'", brokerID, catalogServiceID))
							Expect(serviceOffering.Length().Equal(1))
							serviceOfferingID = serviceOffering.First().Object().Value("id").String().Raw()
							plans := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL,
								fmt.Sprintf("fieldQuery=service_offering_id eq '%s'", serviceOfferingID)).Iter()

							for _, plan := range plans {
								planID := plan.Object().Value("id").String().Raw()
								planIDsForService = append(planIDsForService, planID)
							}

							s, err := sjson.Delete(string(brokerServer.Catalog), "services.0")
							Expect(err).ShouldNot(HaveOccurred())
							brokerServer.Catalog = SBCatalog(s)
						})

						Context("with no existing service instances", func() {
							It("is no longer returned by the Services and Plans API", func() {
								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
									WithJSON(Object{}).
									Expect().
									Status(http.StatusOK)

								ctx.SMWithOAuth.List(web.ServiceOfferingsURL).NotContains(serviceOfferingID)
								ctx.SMWithOAuth.List(web.ServicePlansURL).NotContains(planIDsForService)

								assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
							})

							It("is not returned from the repository as part of the brokers catalog field", func() {
								assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, string(brokerServer.Catalog))
							})
						})

						Context("with existing service instances", func() {
							var serviceInstances []*types.ServiceInstance

							BeforeEach(func() {
								serviceInstances = make([]*types.ServiceInstance, 0)
								for _, planID := range planIDsForService {
									serviceInstance := CreateInstanceInPlatformForPlan(ctx, ctx.TestPlatform.ID, planID, false)
									serviceInstances = append(serviceInstances, serviceInstance)
								}
							})

							AfterEach(func() {
								for _, serviceInstance := range serviceInstances {
									err := DeleteInstance(ctx, serviceInstance.ID, serviceInstance.ServicePlanID)
									Expect(err).ToNot(HaveOccurred())
								}
							})

							It("should return 400 with user-friendly message", func() {
								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
									WithJSON(Object{}).
									Expect().
									Status(http.StatusConflict).
									JSON().Object().
									Value("error").String().Contains("ExistingReferenceEntity")

								ctx.SMWithOAuth.GET(web.ServiceOfferingsURL + "/" + serviceOfferingID).
									Expect().
									Status(http.StatusOK).Body().NotEmpty()

								servicePlans := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("id in ('%s')", strings.Join(planIDsForService, "','")))
								servicePlans.NotEmpty()
								servicePlans.Length().Equal(len(planIDsForService))
							})
						})
					})

					Context("when an existing service offering is modified", func() {
						Context("when catalog service id is modified but the catalog name is not", func() {
							var expectedCatalog string

							BeforeEach(func() {
								catalog, err := sjson.Set(string(brokerServer.Catalog), "services.0.id", "new-id")
								Expect(err).ToNot(HaveOccurred())

								expectedCatalog = catalog

								brokerServer.Catalog = SBCatalog(catalog)
							})

							It("returns 200", func() {
								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).WithJSON(postBrokerRequestWithNoLabels).
									Expect().
									Status(http.StatusOK)

								assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
							})

							Specify("the catalog before the modification is returned by the repository", func() {
								assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, expectedCatalog)

							})
						})

						Context("when both catalog service id and service plan id are modified but the catalog names are not", func() {
							var expectedCatalog string

							BeforeEach(func() {
								catalog, err := sjson.Set(string(brokerServer.Catalog), "services.0.id", "new-svc-id")
								Expect(err).ToNot(HaveOccurred())

								catalog, err = sjson.Set(string(brokerServer.Catalog), "services.0.plans.0.id", "new-plan-id")
								Expect(err).ToNot(HaveOccurred())

								expectedCatalog = catalog

								brokerServer.Catalog = SBCatalog(catalog)
							})

							It("returns 200", func() {
								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).WithJSON(postBrokerRequestWithNoLabels).
									Expect().
									Status(http.StatusOK)

								assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
							})

							Specify("the catalog before the modification is returned by the repository", func() {
								assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, expectedCatalog)

							})
						})

						Context("when catalog service id is removed", func() {
							verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.id")
						})

						Context("when catalog service name is removed", func() {
							verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.name")
						})

						Context("when catalog service description is removed", func() {
							verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusOK)
							}, true, "services.0.description")
						})

						Context("when tags are invalid json", func() {
							verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.tags", "invalidddd")
						})

						Context("when requires is invalid json", func() {
							verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.requires", "{invalid")
						})

						Context("when metadata is invalid json", func() {
							verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.metadata", "{invalid")
						})
					})

					Context("when a new service plan is added", func() {
						var anotherPlanID string
						var serviceOfferingID string

						BeforeEach(func() {
							anotherPlan := JSONToMap(GeneratePaidTestPlan())
							anotherPlanID = anotherPlan["id"].(string)
							Expect(anotherPlan).ToNot(BeEmpty())
							catalogServiceID := gjson.Get(string(brokerServer.Catalog), "services.0.id").Str
							Expect(catalogServiceID).ToNot(BeEmpty())

							serviceOfferings := ctx.SMWithOAuth.List(web.ServiceOfferingsURL).Iter()

							for _, so := range serviceOfferings {
								sbID := so.Object().Value("broker_id").String().Raw()
								Expect(sbID).ToNot(BeEmpty())

								catalogID := so.Object().Value("catalog_id").String().Raw()
								Expect(catalogID).ToNot(BeEmpty())

								if catalogID == catalogServiceID && sbID == brokerID {
									serviceOfferingID = so.Object().Value("id").String().Raw()
									Expect(catalogServiceID).ToNot(BeEmpty())
									break
								}
							}
							s, err := sjson.Set(string(brokerServer.Catalog), "services.0.plans.2", anotherPlan)
							Expect(err).ShouldNot(HaveOccurred())
							brokerServer.Catalog = SBCatalog(s)
						})

						It("is returned from the Plans API associated with the correct service offering", func() {
							ctx.SMWithOAuth.List(web.ServicePlansURL).
								Path("$[*].catalog_id").Array().NotContains(anotherPlanID)

							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
								WithJSON(Object{}).
								Expect().
								Status(http.StatusOK)

							jsonResp := ctx.SMWithOAuth.List(web.ServicePlansURL)
							jsonResp.Path("$[*].catalog_id").Array().Contains(anotherPlanID)
							jsonResp.Path("$[*].service_offering_id").Array().Contains(serviceOfferingID)

							assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
						})

						It("is returned from the repository as part of the brokers catalog field", func() {
							assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, string(brokerServer.Catalog))

						})
					})

					Context("when an existing service plan is removed", func() {
						var removedPlanCatalogID string

						BeforeEach(func() {
							removedPlanCatalogID = gjson.Get(string(brokerServer.Catalog), "services.0.plans.0.id").Str
							Expect(removedPlanCatalogID).ToNot(BeEmpty())
							s, err := sjson.Delete(string(brokerServer.Catalog), "services.0.plans.0")
							Expect(err).ShouldNot(HaveOccurred())
							brokerServer.Catalog = SBCatalog(s)
						})

						Context("with no existing service instances", func() {
							It("is no longer returned by the Plans API", func() {
								ctx.SMWithOAuth.List(web.ServicePlansURL).
									Path("$[*].catalog_id").Array().Contains(removedPlanCatalogID)

								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
									WithJSON(Object{}).
									Expect().
									Status(http.StatusOK)

								ctx.SMWithOAuth.List(web.ServicePlansURL).
									Path("$[*].catalog_id").Array().NotContains(removedPlanCatalogID)

								assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
							})

							It("is not returned from the repository as part of the brokers catalog field", func() {
								assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, string(brokerServer.Catalog))
							})
						})

						Context("with existing service instances", func() {
							var serviceInstance *types.ServiceInstance

							BeforeEach(func() {
								removedPlanID := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_id eq '%s'", removedPlanCatalogID)).
									First().Object().Value("id").String().Raw()

								serviceInstance = CreateInstanceInPlatformForPlan(ctx, ctx.TestPlatform.ID, removedPlanID, false)

							})

							AfterEach(func() {
								err := DeleteInstance(ctx, serviceInstance.ID, serviceInstance.ServicePlanID)
								Expect(err).ToNot(HaveOccurred())
							})

							It("should return 400 with user-friendly message", func() {
								ctx.SMWithOAuth.List(web.ServicePlansURL).
									Path("$[*].catalog_id").Array().Contains(removedPlanCatalogID)

								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
									WithJSON(Object{}).
									Expect().
									Status(http.StatusConflict).
									JSON().Object().
									Value("error").String().Contains("ExistingReferenceEntity")

								ctx.SMWithOAuth.List(web.ServicePlansURL).
									Path("$[*].catalog_id").Array().Contains(removedPlanCatalogID)
							})
						})
					})

					Context("when an existing service plan is modified", func() {
						Context("when catalog service plan id is modified but the catalog name is not", func() {
							var expectedCatalog string

							BeforeEach(func() {
								catalog, err := sjson.Set(string(brokerServer.Catalog), "services.0.plans.0.id", "new-id")
								Expect(err).ToNot(HaveOccurred())

								expectedCatalog = catalog

								brokerServer.Catalog = SBCatalog(catalog)
							})

							It("returns 200", func() {
								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).WithJSON(postBrokerRequestWithNoLabels).
									Expect().
									Status(http.StatusOK)

								assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
							})

							Specify("the catalog before the modification is returned by the repository", func() {
								assertRepositoryReturnsExpectedCatalogAfterPatching(brokerID, expectedCatalog)

							})
						})

						Context("when catalog plan id is removed", func() {
							verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.plans.0.id")
						})

						Context("when catalog plan name is removed", func() {
							verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.plans.0.name")
						})

						Context("when catalog plan description is removed", func() {
							verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.plans.0.description")
						})

						Context("when schemas is invalid json", func() {
							verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.plans.0.schemas", "{invalid")
						})

						Context("when metadata is invalid json", func() {
							verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
								r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
							}, false, "services.0.plans.0.metadata", []byte(`{invalid`))
						})

						Context("when a new service offering with new plans with empty description is added", func() {
							var anotherServiceID string

							BeforeEach(func() {

								planStr := GeneratePaidTestPlan()
								planStr, err := sjson.Delete(planStr, "description")
								Expect(err).ToNot(HaveOccurred())
								anotherPlan := JSONToMap(planStr)

								anotherServiceWithAnotherPlan, err := sjson.Set(GenerateTestServiceWithPlans(), "plans.-1", anotherPlan)
								Expect(err).ShouldNot(HaveOccurred())

								anotherService := JSONToMap(anotherServiceWithAnotherPlan)
								anotherServiceID = anotherService["id"].(string)
								Expect(anotherServiceID).ToNot(BeEmpty())

								catalog, err := sjson.Set(string(brokerServer.Catalog), "services.-1", anotherService)
								Expect(err).ShouldNot(HaveOccurred())

								brokerServer.Catalog = SBCatalog(catalog)
							})

							It("400 bad request is returned", func() {
								ctx.SMWithOAuth.List(web.ServiceOfferingsURL).
									Path("$[*].catalog_id").Array().NotContains(anotherServiceID)
								ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).
									WithJSON(Object{}).
									Expect().
									Status(http.StatusBadRequest)

								assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
							})

						})

					})

				})

				Describe("Labelled", func() {
					var id string
					var patchLabels []types.LabelChange
					var patchLabelsBody map[string]interface{}
					changedLabelKey := "label_key"
					changedLabelValues := []string{"label_value1", "label_value2"}
					operation := types.AddLabelOperation
					BeforeEach(func() {
						patchLabels = []types.LabelChange{}
					})
					JustBeforeEach(func() {
						patchLabelsBody = make(map[string]interface{})
						patchLabels = append(patchLabels, types.LabelChange{
							Operation: operation,
							Key:       changedLabelKey,
							Values:    changedLabelValues,
						})
						patchLabelsBody["labels"] = patchLabels

						id = ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
							WithJSON(postBrokerRequestWithLabels).
							Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
					})

					Context("Add new label", func() {
						It("Should return 200", func() {
							label := types.Labels{changedLabelKey: changedLabelValues}
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().Object().Value("labels").Object().ContainsMap(label)
						})
					})

					Context("Add label with existing key and value", func() {
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK)

							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK)
						})
					})

					Context("Add new label value", func() {
						BeforeEach(func() {
							operation = types.AddLabelValuesOperation
							changedLabelKey = "cluster_id"
							changedLabelValues = []string{"new-label-value"}
						})
						It("Should return 200", func() {
							var labelValuesObj []interface{}
							for _, val := range changedLabelValues {
								labelValuesObj = append(labelValuesObj, val)
							}
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels").Object().Values().Path("$[*][*]").Array().Contains(labelValuesObj...)
						})
					})

					Context("Add new label value to a non-existing label", func() {
						BeforeEach(func() {
							operation = types.AddLabelValuesOperation
							changedLabelKey = "cluster_id_new"
							changedLabelValues = []string{"new-label-value"}
						})
						It("Should return 200", func() {
							var labelValuesObj []interface{}
							for _, val := range changedLabelValues {
								labelValuesObj = append(labelValuesObj, val)
							}

							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels").Object().Values().Path("$[*][*]").Array().Contains(labelValuesObj...)
						})
					})

					Context("Add duplicate label value", func() {
						BeforeEach(func() {
							operation = types.AddLabelValuesOperation
							changedLabelKey = "cluster_id"
							values := labels["cluster_id"].([]interface{})
							changedLabelValues = []string{values[0].(string)}
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK)
						})
					})

					Context("Remove a label", func() {
						BeforeEach(func() {
							operation = types.RemoveLabelOperation
							changedLabelKey = "cluster_id"
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels").Object().Keys().NotContains(changedLabelKey)
						})
					})

					Context("Remove a label and providing no key", func() {
						BeforeEach(func() {
							operation = types.RemoveLabelOperation
							changedLabelKey = ""
						})
						It("Should return 400", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest)
						})
					})

					Context("Remove a label key which does not exist", func() {
						BeforeEach(func() {
							operation = types.RemoveLabelOperation
							changedLabelKey = "non-existing-ey"
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK)
						})
					})

					Context("Remove label values and providing a single value", func() {
						var valueToRemove string
						BeforeEach(func() {
							operation = types.RemoveLabelValuesOperation
							changedLabelKey = "cluster_id"
							valueToRemove = labels[changedLabelKey].([]interface{})[0].(string)
							changedLabelValues = []string{valueToRemove}
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels[*]").Array().NotContains(valueToRemove)
						})
					})

					Context("Remove label values and providing multiple values", func() {
						var valuesToRemove []string
						BeforeEach(func() {
							operation = types.RemoveLabelValuesOperation
							changedLabelKey = "org_id"
							val1 := labels[changedLabelKey].([]interface{})[0].(string)
							val2 := labels[changedLabelKey].([]interface{})[1].(string)
							valuesToRemove = []string{val1, val2}
							changedLabelValues = valuesToRemove
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels[*]").Array().NotContains(valuesToRemove)
						})
					})

					Context("Remove all label values for a key", func() {
						var valuesToRemove []string
						BeforeEach(func() {
							operation = types.RemoveLabelValuesOperation
							changedLabelKey = "cluster_id"
							labelValues := labels[changedLabelKey].([]interface{})
							for _, val := range labelValues {
								valuesToRemove = append(valuesToRemove, val.(string))
							}
							changedLabelValues = valuesToRemove
						})
						It("Should return 200 with this key gone", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels").Object().Keys().NotContains(changedLabelKey)
						})
					})

					Context("Remove label values and not providing value to remove", func() {
						BeforeEach(func() {
							operation = types.RemoveLabelValuesOperation
							changedLabelValues = []string{}
						})
						It("Should return 400", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest)
						})
					})

					Context("Remove label value which does not exist", func() {
						BeforeEach(func() {
							operation = types.RemoveLabelValuesOperation
							changedLabelKey = "cluster_id"
							changedLabelValues = []string{"non-existing-value"}
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK)
						})
					})
				})

				Context("when broker catalog is not changed", func() {
					var (
						brokerID string
					)
					BeforeEach(func() {
						brokerID = ctx.RegisterBroker().Broker.ID
					})

					It("transitive resources should only be updated", func() {
						location := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).WithQuery(web.QueryParamAsync, "true").WithJSON(common.Object{}).Expect().Status(http.StatusAccepted).Header("Location").Raw()
						common.VerifyOperationExists(ctx, location, common.OperationExpectations{
							State:        types.SUCCEEDED,
							Category:     types.UPDATE,
							ResourceType: types.ServiceBrokerType,
						})
						transitive := ctx.SMWithOAuth.GET(location).
							Expect().Status(http.StatusOK).JSON().Object().Value("transitive_resources").Array().Iter()
						for _, r := range transitive {
							if r.Object().Value("type").String().Raw() != types.NotificationType.String() {
								r.Object().Value("operation_type").String().Equal(string(types.UPDATE))
							}
							r.Object().Value("operation_type").String().NotEqual(string(types.DELETE))
						}
					})
				})

				Context("when broker catalog is changed", func() {
					var (
						brokerID     string
						brokerServer *common.BrokerServer
						catalog      common.SBCatalog
					)
					BeforeEach(func() {
						catalog = common.NewRandomSBCatalog()
						testContext := ctx.RegisterBrokerWithCatalog(catalog)
						brokerID = testContext.Broker.ID
						brokerServer = testContext.Broker.BrokerServer
					})

					It("transitive resources should contain deleted plans", func() {
						catalog.RemovePlan(0, 0)
						brokerServer.Catalog = catalog

						location := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+brokerID).WithQuery(web.QueryParamAsync, "true").WithJSON(common.Object{}).Expect().Status(http.StatusAccepted).Header("Location").Raw()
						common.VerifyOperationExists(ctx, location, common.OperationExpectations{
							State:        types.SUCCEEDED,
							Category:     types.UPDATE,
							ResourceType: types.ServiceBrokerType,
						})
						transitive := ctx.SMWithOAuth.GET(location).
							Expect().Status(http.StatusOK).JSON().Object().Value("transitive_resources").Array().Iter()
						deletedCount := 0
						for _, r := range transitive {
							if r.Object().Value("operation_type").String().Raw() == string(types.DELETE) {
								deletedCount++
							}
						}
						Expect(deletedCount).To(Equal(1))
					})
				})

				Context("Instance Sharing", func() {
					var (
						brokerServer *common.BrokerServer
						catalog      common.SBCatalog
					)
					When("the new catalog does not contain a shareable plan", func() {
						BeforeEach(func() {
							catalog, _, _, _, _, _ = common.NewShareableCatalog()
							testContext := ctx.RegisterBrokerWithCatalog(catalog)
							brokerServer = testContext.Broker.BrokerServer
						})
						It("removes the reference plan, when new catalog does not contain shareable plans", func() {
							catalog = common.NewRandomSBCatalog()
							testContext := ctx.RegisterBrokerWithCatalog(catalog)
							//brokerID = testContext.Broker.ID
							brokerServer = testContext.Broker.BrokerServer

							Expect(strings.Contains(string(brokerServer.Catalog), instance_sharing.ReferencePlanName)).To(Equal(false))
						})
					})
					When("changing the supportsInstanceSharing property of an existing plan", func() {
						var plan1, plan2, plan3 string
						var shareableValue bool
						var testContext *BrokerUtils
						BeforeEach(func() {
							catalog, _, _, plan1, plan2, plan3 = common.NewShareableCatalog()
							testContext = ctx.RegisterBrokerWithCatalog(catalog)
							brokerServer = testContext.Broker.BrokerServer
						})
						When("the shareable plan not in use", func() {
							var planID1, planID3 string
							var referencePlan1, referencePlan2 *types.ServicePlan
							BeforeEach(func() {
								// validate plans has shareable status
								path := fmt.Sprintf("metadata.%s", instance_sharing.SupportsInstanceSharingKey)
								shareableValue = gjson.Get(plan1, path).Bool()
								Expect(shareableValue).To(Equal(true))
								shareableValue = gjson.Get(plan2, path).Bool()
								Expect(shareableValue).To(Equal(true))
								shareableValue = gjson.Get(plan3, path).Bool()
								Expect(shareableValue).To(Equal(true))
								// validate reference plan exists
								// reference of service_1:
								planID1 := gjson.Get(plan1, "id").String()
								referencePlan1 := GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID1)
								Expect(referencePlan1).NotTo(Equal(nil))
								// reference of service 2:
								planID3 := gjson.Get(plan3, "id").String()
								referencePlan2 := GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID3)
								Expect(referencePlan2).NotTo(Equal(nil))

								// set catalog as support instance sharing false
								metadataPathPlan1 := fmt.Sprintf("services.0.plans.0.metadata.%s", instance_sharing.SupportsInstanceSharingKey)
								newCatalogBytes, _ := sjson.SetBytes([]byte(brokerServer.Catalog), metadataPathPlan1, false)
								newCatalogBytes, _ = sjson.SetBytes(newCatalogBytes, fmt.Sprintf("services.0.plans.1.metadata.%s", instance_sharing.SupportsInstanceSharingKey), false)
								newCatalogBytes, _ = sjson.SetBytes(newCatalogBytes, fmt.Sprintf("services.1.plans.0.metadata.%s", instance_sharing.SupportsInstanceSharingKey), false)
								brokerServer.Catalog = SBCatalog(newCatalogBytes)
							})
							AfterEach(func() {
								// validate reference plan does not exists
								// reference of service_1:
								planID1 = gjson.Get(plan1, "id").String()
								referencePlan1 = GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID1)
								Expect(referencePlan1).To(BeNil())
								// reference of service 2:
								planID3 = gjson.Get(plan3, "id").String()
								referencePlan2 = GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID3)
								Expect(referencePlan2).To(BeNil())

							})
							It("removes the reference plan when changing the supportsInstanceSharing to 'false'", func() {

								// update broker
								location := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+testContext.Broker.ID).
									WithQuery(web.QueryParamAsync, "true").
									WithJSON(common.Object{}).
									Expect().Status(http.StatusAccepted).Header("Location").Raw()
								common.VerifyOperationExists(ctx, location, common.OperationExpectations{
									State:        types.SUCCEEDED,
									Category:     types.UPDATE,
									ResourceType: types.ServiceBrokerType,
								})

								Expect(strings.Contains(string(brokerServer.Catalog), instance_sharing.ReferencePlanName)).To(Equal(false))

							})
						})
						When("the shareable plan has reference instance", func() {
							var planID1 string
							var referencePlan1 *types.ServicePlan
							var sharedInstance *types.ServiceInstance
							var referenceInstance *types.ServiceInstance
							BeforeEach(func() {
								// validate plans has shareable status
								path := fmt.Sprintf("metadata.%s", instance_sharing.SupportsInstanceSharingKey)
								shareableValue = gjson.Get(plan1, path).Bool()
								Expect(shareableValue).To(Equal(true))
								// validate reference plan exists
								// reference of service_1:
								planID1 = gjson.Get(plan1, "id").String()
								referencePlan1 = GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID1)
								Expect(referencePlan1).NotTo(Equal(nil))

								sharedPlan := GetPlanByKey(ctx, "catalog_id", planID1)
								sharedInstance = CreateInstanceInPlatformForPlan(ctx, ctx.TestPlatform.ID, sharedPlan.ID, true)
								referenceInstance = CreateReferenceInstanceInPlatform(ctx, ctx.TestPlatform.ID, referencePlan1.ID, sharedInstance.ID)
								//set catalog as support instance sharing false
								newCatalogBytes, _ := sjson.SetBytes([]byte(brokerServer.Catalog), fmt.Sprintf("services.0.plans.0.metadata.%s", instance_sharing.SupportsInstanceSharingKey), false)
								brokerServer.Catalog = SBCatalog(newCatalogBytes)
							})
							AfterEach(func() {
								Expect(strings.Contains(string(brokerServer.Catalog), instance_sharing.ReferencePlanName)).To(Equal(false))
								// validate reference plan does not exists
								// reference of service_1:
								planID1 = gjson.Get(plan1, "id").String()
								referencePlan1 = GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID1)
								Expect(referencePlan1).NotTo(BeNil())
								DeleteInstance(ctx, referenceInstance.ID, referenceInstance.ServicePlanID)
								DeleteInstance(ctx, sharedInstance.ID, sharedInstance.ServicePlanID)
							})
							It("fails removing the reference plan after changing the supportsInstanceSharing to 'false' (for planID1) due to existing shared instance of a plan", func() {
								// update broker
								location := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+testContext.Broker.ID).
									WithQuery(web.QueryParamAsync, "true").
									WithJSON(common.Object{}).
									Expect().Status(http.StatusAccepted).Header("Location").Raw()
								common.VerifyOperationExists(ctx, location, common.OperationExpectations{
									State:        types.FAILED,
									Category:     types.UPDATE,
									ResourceType: types.ServiceBrokerType,
								})
							})
						})
						When("the shareable plan has shared instance", func() {
							var planID1 string
							var referencePlan1 *types.ServicePlan
							var sharedInstance *types.ServiceInstance
							BeforeEach(func() {
								// validate plans has shareable status
								path := fmt.Sprintf("metadata.%s", instance_sharing.SupportsInstanceSharingKey)
								shareableValue = gjson.Get(plan1, path).Bool()
								Expect(shareableValue).To(Equal(true))
								// validate reference plan exists
								// reference of service_1:
								planID1 = gjson.Get(plan1, "id").String()
								referencePlan1 = GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID1)
								Expect(referencePlan1).NotTo(Equal(nil))

								sharedPlan := GetPlanByKey(ctx, "catalog_id", planID1)
								sharedInstance = CreateInstanceInPlatformForPlan(ctx, ctx.TestPlatform.ID, sharedPlan.ID, true)
								//set catalog as support instance sharing false
								newCatalogBytes, _ := sjson.SetBytes([]byte(brokerServer.Catalog), fmt.Sprintf("services.0.plans.0.metadata.%s", instance_sharing.SupportsInstanceSharingKey), false)
								brokerServer.Catalog = SBCatalog(newCatalogBytes)
							})
							AfterEach(func() {
								Expect(strings.Contains(string(brokerServer.Catalog), instance_sharing.ReferencePlanName)).To(Equal(false))
								// validate reference plan does not exists
								// reference of service_1:
								planID1 = gjson.Get(plan1, "id").String()
								referencePlan1 = GetReferencePlanOfExistingPlan(ctx, "catalog_id", planID1)
								Expect(referencePlan1).NotTo(BeNil())
								DeleteInstance(ctx, sharedInstance.ID, sharedInstance.ServicePlanID)
							})
							It("fails removing the reference plan after changing the supportsInstanceSharing to 'false' (for planID1) due to existing shared instance of a plan", func() {
								// update broker
								location := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+testContext.Broker.ID).
									WithQuery(web.QueryParamAsync, "true").
									WithJSON(common.Object{}).
									Expect().Status(http.StatusAccepted).Header("Location").Raw()
								common.VerifyOperationExists(ctx, location, common.OperationExpectations{
									State:        types.FAILED,
									Category:     types.UPDATE,
									ResourceType: types.ServiceBrokerType,
								})
							})
						})
					})
					When("a catalog without a shareable plan is being updated with a new shareable plan", func() {
						var testContext *BrokerUtils
						BeforeEach(func() {
							catalog = common.NewRandomSBCatalog()
							testContext = ctx.RegisterBrokerWithCatalog(catalog)
							brokerServer = testContext.Broker.BrokerServer
							brokerID = testContext.Broker.ID
							// set catalog as support instance sharing true
							metadataPathPlan1 := fmt.Sprintf("services.0.plans.0.metadata.%s", instance_sharing.SupportsInstanceSharingKey)
							newCatalogBytes, _ := sjson.SetBytes([]byte(brokerServer.Catalog), metadataPathPlan1, true)
							brokerServer.Catalog = SBCatalog(newCatalogBytes)
						})
						It("should generate a reference plan", func() {
							// update broker
							resp := ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL+"/"+testContext.Broker.ID).
								WithQuery(web.QueryParamAsync, "true").
								WithJSON(common.Object{}).
								Expect().Status(http.StatusAccepted)
							common.VerifyOperationExists(ctx, resp.Header("Location").Raw(), common.OperationExpectations{
								State:        types.SUCCEEDED,
								Category:     types.UPDATE,
								ResourceType: types.ServiceBrokerType,
							})

							byID := query.ByField(query.EqualsOperator, "id", brokerID)
							brokerFromDB, err := repository.Get(context.TODO(), types.ServiceBrokerType, byID)
							Expect(err).ToNot(HaveOccurred())
							catalogJSON := string(brokerFromDB.(*types.ServiceBroker).Catalog)
							referencePlan := gjson.Get(catalogJSON, "services.0.plans.4").String()
							Expect(referencePlan).To(ContainSubstring(instance_sharing.ReferencePlanName))
							Expect(referencePlan).To(ContainSubstring(instance_sharing.ReferencePlanDescription))
						})
					})
				})
			})

			Describe("DELETE", func() {
				Context("with existing service instances to some broker plan", func() {
					var (
						brokerID        string
						serviceInstance *types.ServiceInstance
					)

					BeforeEach(func() {
						brokerID, serviceInstance = CreateInstanceInPlatform(ctx, ctx.TestPlatform.ID)
					})

					AfterEach(func() {
						err := DeleteInstance(ctx, serviceInstance.ID, serviceInstance.ServicePlanID)
						Expect(err).ToNot(HaveOccurred())
					})

					It("should return 400 with user-friendly message", func() {
						ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).
							Expect().
							Status(http.StatusConflict).
							JSON().Object().
							Value("error").String().Contains("ExistingReferenceEntity")
					})
				})

				Context("when attempting async bulk delete", func() {
					It("should return 400", func() {
						ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL).
							WithQuery("fieldQuery", "id in ('id1','id2','id3')").
							WithQuery("async", "true").
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().
							Value("description").String().Contains("Only one resource can be deleted asynchronously at a time")
					})
				})

				Context("when there are transitive resources", func() {
					var brokerID string
					BeforeEach(func() {
						brokerID = ctx.RegisterBroker().Broker.ID
					})

					It("should keep them in the operation", func() {
						ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect().Status(http.StatusOK)
						operation, err := ctx.SMRepository.Get(context.Background(), types.OperationType,
							query.ByField(query.EqualsOperator, "resource_id", brokerID),
							query.ByField(query.EqualsOperator, "type", string(types.DELETE)))
						Expect(err).ShouldNot(HaveOccurred())
						Expect(operation.(*types.Operation).TransitiveResources).Should(HaveLen(1))
					})
				})
			})
		})
	},
})

func blueprint(setNullFieldsValues bool) func(ctx *TestContext, auth *SMExpect, async bool) Object {
	return func(ctx *TestContext, auth *SMExpect, async bool) Object {
		brokerJSON := GenerateRandomBroker()

		if !setNullFieldsValues {
			delete(brokerJSON, "description")
		}

		var obj map[string]interface{}
		resp := auth.POST(web.ServiceBrokersURL).WithQuery("async", strconv.FormatBool(async)).WithJSON(brokerJSON).Expect()
		if async {
			resp.Status(http.StatusAccepted)
		} else {
			resp.Status(http.StatusCreated)
		}

		id, _ := VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
			Category:          types.CREATE,
			State:             types.SUCCEEDED,
			ResourceType:      types.ServiceBrokerType,
			Reschedulable:     false,
			DeletionScheduled: false,
		})

		obj = VerifyResourceExists(ctx.SMWithOAuth, ResourceExpectations{
			ID:    id,
			Type:  types.ServiceBrokerType,
			Ready: true,
		}).Raw()

		return obj
	}
}

type labeledBroker Object

func (b labeledBroker) SetLabels(labels Object) {
	b["labels"] = labels
}
