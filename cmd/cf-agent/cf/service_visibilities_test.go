package cf_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/agent/reconcile"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/cmd/cf-agent/cf"
	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client Service Plan Visibilities", func() {
	const orgGUID = "testorgguid"

	var (
		ccServer                *ghttp.Server
		client                  *cf.PlatformClient
		ctx                     context.Context
		generatedCFBrokers      []*cfclient.ServiceBroker
		generatedCFServices     map[string][]*cfclient.Service
		generatedCFPlans        map[string][]*cfclient.ServicePlan
		generatedCFVisibilities map[string]*cfclient.ServicePlanVisibility
		expectedCFVisibiltiies  map[string]*platform.Visibility

		maxAllowedParallelRequests int
		parallelRequestsCounter    int
		parallelRequestsMutex      sync.Mutex
	)

	generateCFBrokers := func(count int) []*cfclient.ServiceBroker {
		brokers := make([]*cfclient.ServiceBroker, 0)
		for i := 0; i < count; i++ {
			UUID, err := uuid.NewV4()
			Expect(err).ShouldNot(HaveOccurred())
			brokerGuid := "broker-" + UUID.String()
			brokerName := fmt.Sprintf("broker%d", i)
			brokers = append(brokers, &cfclient.ServiceBroker{
				Guid: brokerGuid,
				Name: reconcile.DefaultProxyBrokerPrefix + brokerName + "-" + brokerGuid,
			})
		}
		return brokers
	}

	generateCFServices := func(brokers []*cfclient.ServiceBroker, count int) map[string][]*cfclient.Service {
		services := make(map[string][]*cfclient.Service)
		for _, broker := range brokers {
			for i := 0; i < count; i++ {
				UUID, err := uuid.NewV4()
				Expect(err).ShouldNot(HaveOccurred())

				serviceGUID := "service-" + UUID.String()
				services[broker.Guid] = append(services[broker.Guid], &cfclient.Service{
					Guid:              serviceGUID,
					ServiceBrokerGuid: broker.Guid,
				})
			}
		}
		return services
	}

	generateCFPlans := func(servicesMap map[string][]*cfclient.Service, plansToGenrate, publicPlansToGenerate int) map[string][]*cfclient.ServicePlan {
		plans := make(map[string][]*cfclient.ServicePlan)

		for _, services := range servicesMap {
			for _, service := range services {
				for i := 0; i < plansToGenrate; i++ {
					UUID, err := uuid.NewV4()
					Expect(err).ShouldNot(HaveOccurred())
					plans[service.Guid] = append(plans[service.Guid], &cfclient.ServicePlan{
						Guid:        "planGUID-" + UUID.String(),
						UniqueId:    "planCatalogGUID-" + UUID.String(),
						ServiceGuid: service.Guid,
					})
				}

				for i := 0; i < publicPlansToGenerate; i++ {
					UUID, err := uuid.NewV4()
					Expect(err).ShouldNot(HaveOccurred())
					plans[service.Guid] = append(plans[service.Guid], &cfclient.ServicePlan{
						Guid:        "planGUID-" + UUID.String(),
						UniqueId:    "planCatalogGUID-" + UUID.String(),
						ServiceGuid: service.Guid,
						Public:      true,
					})
				}
			}
		}
		return plans
	}

	generateCFVisibilities := func(plansMap map[string][]*cfclient.ServicePlan) (map[string]*cfclient.ServicePlanVisibility, map[string]*platform.Visibility) {
		visibilities := make(map[string]*cfclient.ServicePlanVisibility)
		expectedVisibilities := make(map[string]*platform.Visibility, 0)
		for _, plans := range plansMap {
			for _, plan := range plans {
				visibilityGuid := "cfVisibilityForPlan_" + plan.Guid
				var brokerName string
				for _, services := range generatedCFServices {
					for _, service := range services {
						if service.Guid == plan.ServiceGuid {
							brokerName = ""
							for _, cfBroker := range generatedCFBrokers {
								if cfBroker.Guid == service.ServiceBrokerGuid {
									brokerName = cfBroker.Name
								}
							}
						}
					}
				}
				Expect(brokerName).ToNot(BeEmpty())

				if !plan.Public {
					visibilities[plan.Guid] = &cfclient.ServicePlanVisibility{
						ServicePlanGuid:  plan.Guid,
						ServicePlanUrl:   "http://example.com",
						Guid:             visibilityGuid,
						OrganizationGuid: orgGUID,
					}

					expectedVisibilities[plan.Guid] = &platform.Visibility{
						Public:             false,
						CatalogPlanID:      plan.UniqueId,
						PlatformBrokerName: brokerName,
						Labels: map[string]string{
							client.VisibilityScopeLabelKey(): orgGUID,
						},
					}
				} else {
					expectedVisibilities[plan.Guid] = &platform.Visibility{
						Public:             true,
						CatalogPlanID:      plan.UniqueId,
						PlatformBrokerName: brokerName,
						Labels:             make(map[string]string),
					}
				}
			}
		}

		return visibilities, expectedVisibilities
	}

	parallelRequestsChecker := func(f http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			parallelRequestsMutex.Lock()
			parallelRequestsCounter++
			if parallelRequestsCounter > maxAllowedParallelRequests {
				defer func() {
					parallelRequestsMutex.Lock()
					defer parallelRequestsMutex.Unlock()
					Fail(fmt.Sprintf("Max allowed parallel requests is %d but %d were detected", maxAllowedParallelRequests, parallelRequestsCounter))
				}()

			}
			parallelRequestsMutex.Unlock()
			defer func() {
				parallelRequestsMutex.Lock()
				parallelRequestsCounter--
				parallelRequestsMutex.Unlock()
			}()

			// Simulate a 80ms request
			<-time.After(80 * time.Millisecond)
			f(writer, request)
		}
	}

	parseFilterQuery := func(plansQuery, queryKey string) map[string]bool {
		Expect(plansQuery).ToNot(BeEmpty())

		prefix := queryKey + " IN "
		Expect(plansQuery).To(HavePrefix(prefix))

		plansQuery = strings.TrimPrefix(plansQuery, prefix)
		plans := strings.Split(plansQuery, ",")
		Expect(plans).ToNot(BeEmpty())

		result := make(map[string]bool)
		for _, plan := range plans {
			result[plan] = true
		}
		return result
	}

	writeJSONResponse := func(respStruct interface{}, rw http.ResponseWriter) {
		jsonResponse, err := json.Marshal(respStruct)
		Expect(err).ToNot(HaveOccurred())

		rw.WriteHeader(http.StatusOK)
		rw.Write(jsonResponse)
	}

	getBrokerNames := func(cfBrokers []*cfclient.ServiceBroker) []string {
		names := make([]string, 0, len(cfBrokers))
		for _, cfBroker := range cfBrokers {
			names = append(names, cfBroker.Name)
		}
		return names
	}

	badRequestHandler := func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(`{"error": "Expected"}`))
	}

	setCCBrokersResponse := func(server *ghttp.Server, cfBrokers []*cfclient.ServiceBroker) {
		if cfBrokers == nil {
			server.RouteToHandler(http.MethodGet, "/v2/service_brokers", parallelRequestsChecker(badRequestHandler))
			return
		}
		server.RouteToHandler(http.MethodGet, "/v2/service_brokers", parallelRequestsChecker(func(rw http.ResponseWriter, req *http.Request) {
			filter := parseFilterQuery(req.URL.Query().Get("q"), "name")
			result := make([]cfclient.ServiceBrokerResource, 0, len(filter))
			for _, broker := range cfBrokers {
				if _, found := filter[broker.Name]; found {
					result = append(result, cfclient.ServiceBrokerResource{
						Entity: *broker,
						Meta: cfclient.Meta{
							Guid: broker.Guid,
						},
					})
				}
			}
			response := cfclient.ServiceBrokerResponse{
				Count:     len(result),
				Pages:     1,
				Resources: result,
			}
			writeJSONResponse(response, rw)
		}))
	}

	setCCServicesResponse := func(server *ghttp.Server, cfServices map[string][]*cfclient.Service) {
		if cfServices == nil {
			server.RouteToHandler(http.MethodGet, "/v2/services", parallelRequestsChecker(badRequestHandler))
			return
		}
		server.RouteToHandler(http.MethodGet, "/v2/services", parallelRequestsChecker(func(rw http.ResponseWriter, req *http.Request) {
			filter := parseFilterQuery(req.URL.Query().Get("q"), "service_broker_guid")
			result := make([]cfclient.ServicesResource, 0, len(filter))
			for _, services := range cfServices {
				for _, service := range services {
					if _, found := filter[service.ServiceBrokerGuid]; found {
						result = append(result, cfclient.ServicesResource{
							Entity: *service,
							Meta: cfclient.Meta{
								Guid: service.Guid,
							},
						})
					}
				}
			}
			response := cfclient.ServicesResponse{
				Count:     len(result),
				Pages:     1,
				Resources: result,
			}
			writeJSONResponse(response, rw)
		}))
	}

	setCCPlansResponse := func(server *ghttp.Server, cfPlans map[string][]*cfclient.ServicePlan) {
		if cfPlans == nil {
			server.RouteToHandler(http.MethodGet, "/v2/service_plans", parallelRequestsChecker(badRequestHandler))
			return
		}
		server.RouteToHandler(http.MethodGet, "/v2/service_plans", parallelRequestsChecker(func(rw http.ResponseWriter, req *http.Request) {
			filterQuery := parseFilterQuery(req.URL.Query().Get("q"), "service_guid")
			planResources := make([]cfclient.ServicePlanResource, 0, len(filterQuery))
			for _, plans := range cfPlans {
				for _, plan := range plans {
					if _, found := filterQuery[plan.ServiceGuid]; found {
						planResources = append(planResources, cfclient.ServicePlanResource{
							Entity: *plan,
							Meta: cfclient.Meta{
								Guid: plan.Guid,
							},
						})
					}
				}
			}
			servicePlanResponse := cfclient.ServicePlansResponse{
				Count:     len(planResources),
				Pages:     1,
				Resources: planResources,
			}
			writeJSONResponse(servicePlanResponse, rw)
		}))
	}

	setCCVisibilitiesResponse := func(server *ghttp.Server, cfVisibilities map[string]*cfclient.ServicePlanVisibility) {
		if cfVisibilities == nil {
			server.RouteToHandler(http.MethodGet, "/v2/service_plan_visibilities", parallelRequestsChecker(badRequestHandler))
			return
		}
		server.RouteToHandler(http.MethodGet, "/v2/service_plan_visibilities", parallelRequestsChecker(func(rw http.ResponseWriter, req *http.Request) {
			reqPlans := parseFilterQuery(req.URL.Query().Get("q"), "service_plan_guid")
			visibilityResources := make([]cfclient.ServicePlanVisibilityResource, 0, len(reqPlans))
			for visibilityGuid, visibility := range cfVisibilities {
				if _, found := reqPlans[visibility.ServicePlanGuid]; found {
					visibilityResources = append(visibilityResources, cfclient.ServicePlanVisibilityResource{
						Entity: *visibility,
						Meta: cfclient.Meta{
							Guid: visibilityGuid,
						},
					})
				}
			}
			servicePlanResponse := cfclient.ServicePlanVisibilitiesResponse{
				Count:     len(visibilityResources),
				Pages:     1,
				Resources: visibilityResources,
			}
			writeJSONResponse(servicePlanResponse, rw)
		}))
	}

	createCCServer := func(brokers []*cfclient.ServiceBroker, cfServices map[string][]*cfclient.Service, cfPlans map[string][]*cfclient.ServicePlan, cfVisibilities map[string]*cfclient.ServicePlanVisibility) *ghttp.Server {
		server := fakeCCServer(false)
		setCCBrokersResponse(server, brokers)
		setCCServicesResponse(server, cfServices)
		setCCPlansResponse(server, cfPlans)
		setCCVisibilitiesResponse(server, cfVisibilities)

		return server
	}

	AfterEach(func() {
		if ccServer != nil {
			ccServer.Close()
			ccServer = nil
		}
	})

	BeforeEach(func() {
		ctx = context.TODO()

		generatedCFBrokers = generateCFBrokers(5)
		generatedCFServices = generateCFServices(generatedCFBrokers, 10)
		generatedCFPlans = generateCFPlans(generatedCFServices, 15, 2)
		generatedCFVisibilities, expectedCFVisibiltiies = generateCFVisibilities(generatedCFPlans)

		parallelRequestsCounter = 0
		maxAllowedParallelRequests = 3
	})

	It("is not nil", func() {
		ccServer = createCCServer(generatedCFBrokers, nil, nil, nil)
		_, client = ccClient(ccServer.URL())
		Expect(client.Visibility()).ToNot(BeNil())
	})

	Describe("Get visibilities when visibilities are available", func() {
		BeforeEach(func() {
			ccServer = createCCServer(generatedCFBrokers, generatedCFServices, generatedCFPlans, generatedCFVisibilities)
			_, client = ccClientWithThrottling(ccServer.URL(), maxAllowedParallelRequests)
		})

		Context("for multiple brokers", func() {
			It("should return all visibilities, including ones for public plans", func() {
				platformVisibilities, err := client.GetVisibilitiesByBrokers(ctx, getBrokerNames(generatedCFBrokers))
				Expect(err).ShouldNot(HaveOccurred())

				for _, expectedCFVisibility := range expectedCFVisibiltiies {
					Expect(platformVisibilities).Should(ContainElement(expectedCFVisibility))
				}
			})
		})

		Context("but a single broker", func() {
			It("should return the correct visibilities", func() {
				for _, generatedCFBroker := range generatedCFBrokers {
					brokerGUID := generatedCFBroker.Guid
					platformVisibilities, err := client.GetVisibilitiesByBrokers(ctx, []string{
						generatedCFBroker.Name,
					})
					Expect(err).ShouldNot(HaveOccurred())

					for _, service := range generatedCFServices[brokerGUID] {
						serviceGUID := service.Guid
						for _, plan := range generatedCFPlans[serviceGUID] {
							planGUID := plan.Guid
							expectedVis := expectedCFVisibiltiies[planGUID]
							Expect(platformVisibilities).Should(ContainElement(expectedVis))
						}
					}
				}
			})
		})
	})

	Describe("Get visibilities when cloud controller is not working", func() {
		Context("for getting services", func() {
			BeforeEach(func() {
				ccServer = createCCServer(generatedCFBrokers, nil, nil, nil)
				_, client = ccClientWithThrottling(ccServer.URL(), maxAllowedParallelRequests)
			})

			It("should return error", func() {
				_, err := client.GetVisibilitiesByBrokers(ctx, getBrokerNames(generatedCFBrokers))
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("could not get services from platform"))
			})
		})

		Context("for getting plans", func() {
			BeforeEach(func() {
				ccServer = createCCServer(generatedCFBrokers, generatedCFServices, nil, nil)
				_, client = ccClientWithThrottling(ccServer.URL(), maxAllowedParallelRequests)
			})

			It("should return error", func() {
				_, err := client.GetVisibilitiesByBrokers(ctx, getBrokerNames(generatedCFBrokers))
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("could not get plans from platform"))
			})
		})

		Context("for getting visibilities", func() {
			BeforeEach(func() {
				ccServer = createCCServer(generatedCFBrokers, generatedCFServices, generatedCFPlans, nil)
				_, client = ccClientWithThrottling(ccServer.URL(), maxAllowedParallelRequests)
			})

			It("should return error", func() {
				_, err := client.GetVisibilitiesByBrokers(ctx, getBrokerNames(generatedCFBrokers))
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("could not get visibilities from platform"))
			})
		})
	})
})
