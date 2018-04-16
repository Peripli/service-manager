package osb

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"net/http"
	"net/http/httptest"

	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/Peripli/service-manager/types"
	"github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/go-open-service-broker-client/v2/fake"
	"github.com/pmorie/osb-broker-lib/pkg/broker"
)

var _ = Describe("Logic", func() {
	const (
		testClusterServiceClassName           = "test-service"
		testClusterServicePlanName            = "test-plan"
		testNonbindableClusterServicePlanName = "test-nb-plan"
		testClassExternalID                   = "12345"
		testPlanExternalID                    = "34567"
		testNonbindablePlanExternalID         = "nb34567"
	)
	var (
		fakeBroker              *types.Broker
		fakeBrokerStorage       *storagefakes.FakeBroker
		fakeClient              *fake.FakeClient
		createClientFnCallCount int
		fakeClientCreateFunc    func(_ *v2.ClientConfiguration) (v2.Client, error)
		reactionError           *v2.HTTPStatusCodeError

		logic BusinessLogic
	)

	getTestCatalogResponse := func() *v2.CatalogResponse {
		return &v2.CatalogResponse{
			Services: []v2.Service{
				{
					Name:        testClusterServiceClassName,
					ID:          testClassExternalID,
					Description: "a test service",
					Bindable:    true,
					Plans: []v2.Plan{
						{
							Name:        testClusterServicePlanName,
							Free:        boolPtr(true),
							ID:          testPlanExternalID,
							Description: "a test plan",
						},
						{
							Name:        testNonbindableClusterServicePlanName,
							Free:        boolPtr(true),
							ID:          testNonbindablePlanExternalID,
							Description: "an non-bindable test plan",
							Bindable:    boolPtr(false),
						},
					},
				},
			},
		}
	}

	happyPathConfig := func() fake.FakeClientConfiguration {
		return fake.FakeClientConfiguration{
			CatalogReaction: &fake.CatalogReaction{
				Response: getTestCatalogResponse(),
			},
			ProvisionReaction: &fake.ProvisionReaction{
				Response: &v2.ProvisionResponse{},
			},
			UpdateInstanceReaction: &fake.UpdateInstanceReaction{
				Response: &v2.UpdateInstanceResponse{},
			},
			DeprovisionReaction: &fake.DeprovisionReaction{
				Response: &v2.DeprovisionResponse{},
			},
			BindReaction: &fake.BindReaction{
				Response: &v2.BindResponse{
					Async: false,
					Credentials: map[string]interface{}{
						"host":     "localhost",
						"port":     "8080",
						"username": "admin",
						"password": "admin",
					},
				},
			},
			UnbindReaction: &fake.UnbindReaction{
				Response: &v2.UnbindResponse{},
			},
			PollLastOperationReaction: &fake.PollLastOperationReaction{
				Response: &v2.LastOperationResponse{
					State: v2.StateSucceeded,
				},
			},
			PollBindingLastOperationReaction: &fake.PollBindingLastOperationReaction{
				Response: &v2.LastOperationResponse{
					State: v2.StateSucceeded,
				},
			},
			GetBindingReaction: &fake.GetBindingReaction{
				Response: &v2.GetBindingResponse{
					Credentials: map[string]interface{}{
						"host":     "localhost",
						"port":     "8080",
						"username": "admin",
						"password": "admin",
					},
				},
			},
		}
	}

	assertErrorPropagation := func() {

	}

	assertOSBClientInvoked := func(invocationsCount int) func() {
		return func() {
			Expect(createClientFnCallCount).To(Equal(invocationsCount))
		}
	}

	newFakeClientFunc := func(config fake.FakeClientConfiguration) v2.CreateFunc {
		createClientFnCallCount = 0
		return func(_ *v2.ClientConfiguration) (v2.Client, error) {
			createClientFnCallCount++
			return fake.NewFakeClient(config), nil
		}
	}

	BeforeEach(func() {
		fakeBroker = &types.Broker{
			ID:       "brokerID",
			Name:     "brokerName",
			URL:      "http://localhost:8080/broker",
			User:     "admin",
			Password: "admin",
		}
		fakeClientCreateFunc = newFakeClientFunc(happyPathConfig())
		fakeBrokerStorage = &storagefakes.FakeBroker{}
		logic = BusinessLogic{
			createFunc:    fakeClientCreateFunc,
			brokerStorage: fakeBrokerStorage,
		}
		reactionError = &v2.HTTPStatusCodeError{
			StatusCode:   http.StatusInternalServerError,
			ErrorMessage: strPtr("error message"),
			Description:  strPtr("response description"),
		}
	})

	Describe("GetCatalog", func() {

		var (
			expectedCatalogResponse *v2.CatalogResponse
		)

		callGetCatalog := func(brokerID string) (*v2.CatalogResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/catalog", nil)
			if brokerID != "" {
				request.WithContext(context.WithValue(request.Context(), "brokerID", brokerID))
			}
			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.GetCatalog(context)
			var catalogResp *v2.CatalogResponse
			if response != nil {
				catalogResp = &response.CatalogResponse
			}
			return catalogResp, err
		}

		BeforeEach(func() {
			fakeBrokerStorage.FindReturns(fakeBroker, nil)
			fakeClient.CatalogReaction = &fake.CatalogReaction{
				Response: expectedCatalogResponse,
				Error:    nil,
			}
		})

		It("invokes the OSB client", assertOSBClientInvoked(1))

		It("returns proper catalog response", func() {
			response, err := callGetCatalog("brokerID")
		})

		It("returns no error", func() {
			//_, err := logic.GetCatalog(requestContext)

			//Expect(err).ToNot(HaveOccurred())
		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				// mock to return the error
			})

			It("returns an error", assertErrorPropagation)

			It("does not invoke the OSB client", assertOSBClientInvoked(0))
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				// mock to return the error
			})

			It("returns an error", assertErrorPropagation)

			It("does not invoke the OSB client", assertOSBClientInvoked(0))
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				// mock to return the error
			})

			It("propagates the error", assertErrorPropagation)

			It("does not invoke the OSB client", assertOSBClientInvoked(0))
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				// mock to return the error
			})

			It("invokes the OSB client", assertOSBClientInvoked(1))

			It("propagates the error", assertErrorPropagation)

		})

	})

	Describe("Provision", func() {
		//callProvision := func() {

		//}

	})

	Describe("Desprovision", func() {

	})

	Describe("Last Operation", func() {

	})

	Describe("Bind", func() {

	})

	Describe("Unbind", func() {

	})

	Describe("Update", func() {

	})

	Describe("Validate Broker API Version", func() {
		It("returns no error for latest API version", func() {

		})

		It("returns an error for older API versions", func() {

		})

	})

})

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
