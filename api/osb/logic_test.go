/*
 * Copyright 2018 The Service Manager Authors
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

package osb

import (
	"github.com/Peripli/service-manager/storage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/http/httptest"

	"fmt"

	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/Peripli/service-manager/types"
	"github.com/gorilla/mux"
	"github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/go-open-service-broker-client/v2/fake"
	"github.com/pmorie/osb-broker-lib/pkg/broker"
)

var _ = Describe("Logic", func() {

	var (
		actualErr   error
		expectedErr error

		actionType fake.ActionType

		brokerID string

		fakeBroker        *types.Broker
		fakeBrokerStorage *storagefakes.FakeBroker
		fakeClient        *fake.FakeClient

		createClientFnCallCount int
		fakeClientCreateFunc    func(_ *v2.ClientConfiguration) (v2.Client, error)

		reactionError *v2.HTTPStatusCodeError

		logic BusinessLogic
	)

	newFakeClientFunc := func(client *fake.FakeClient, returnedError error) v2.CreateFunc {
		createClientFnCallCount = 0
		return func(_ *v2.ClientConfiguration) (v2.Client, error) {
			createClientFnCallCount++
			if returnedError != nil {
				return nil, returnedError
			}
			return client, nil
		}
	}

	assertOsbClientCreateFnInvoked := func(invocationsCount int) func() {
		return func() {
			Expect(createClientFnCallCount).To(Equal(invocationsCount))
		}
	}

	assertAllRelevantInvocationsHappened := func() {
		It("invokes find from broker storage", func() {
			Expect(fakeBrokerStorage.GetCallCount()).To(Equal(1))
			id := fakeBrokerStorage.GetArgsForCall(0)
			Expect(id).To(Equal(brokerID))
		})

		It("invokes the OSB client creation function", func() {
			assertOsbClientCreateFnInvoked(1)
		})

		It("invokes a proper client action", func() {
			Expect(len(fakeClient.Actions())).To(Equal(1))
			action := fakeClient.Actions()[0]
			Expect(action.Type).To(Equal(actionType))
		})
	}

	assertBehaviourWhenBrokerIDPathParameterIsMissing := func() {
		It("does not invoke find from broker storage", func() {
			Expect(fakeBrokerStorage.GetCallCount()).To(Equal(0))
		})

		It("does not invoke the OSB client create function", func() {
			assertOsbClientCreateFnInvoked(0)
		})

		It("does not invoke any client action", func() {
			Expect(len(fakeClient.Actions())).To(Equal(0))
		})

		It("returns an error", func() {
			Expect(actualErr).To(HaveOccurred())
		})
	}

	assertBehaviourWhenBrokerNotFoundInStorage := func() {
		It("invokes find from broker storage", func() {
			Expect(fakeBrokerStorage.GetCallCount()).To(Equal(1))
			id := fakeBrokerStorage.GetArgsForCall(0)
			Expect(id).To(Equal(brokerID))
		})

		It("does not invoke the OSB client create function", func() {
			assertOsbClientCreateFnInvoked(0)
		})

		It("does not invoke any client action", func() {
			Expect(len(fakeClient.Actions())).To(Equal(0))
		})

		It("returns an error", func() {
			Expect(actualErr).To(HaveOccurred())
			Expect(actualErr.Error()).To(ContainSubstring(expectedErr.Error()))
		})
	}

	assertBehaviourWhenErrorOccursDuringOsbClientCreation := func() {
		It("invokes find from broker storage", func() {
			Expect(fakeBrokerStorage.GetCallCount()).To(Equal(1))
			id := fakeBrokerStorage.GetArgsForCall(0)
			Expect(id).To(Equal(brokerID))
		})

		It("invokes the OSB client creation function", func() {
			assertOsbClientCreateFnInvoked(1)
		})

		It("does not invoke any client action", func() {
			Expect(len(fakeClient.Actions())).To(Equal(0))
		})

		It("returns an error", func() {
			Expect(actualErr).To(HaveOccurred())
			Expect(actualErr).To(Equal(expectedErr))
		})
	}

	assertBehaviourWhenErrorOccursDuringOsbCall := func() {
		It("invokes find from broker storage", func() {
			Expect(fakeBrokerStorage.GetCallCount()).To(Equal(1))
			id := fakeBrokerStorage.GetArgsForCall(0)
			Expect(id).To(Equal(brokerID))
		})

		It("invokes the OSB client create function", func() {
			assertOsbClientCreateFnInvoked(1)
		})

		It("invokes a proper client action", func() {
			Expect(len(fakeClient.Actions())).To(Equal(1))
			action := fakeClient.Actions()[0]
			Expect(action.Type).To(Equal(actionType))
		})

		It("returns an error", func() {
			Expect(actualErr).To(HaveOccurred())
			Expect(actualErr).To(Equal(expectedErr))
		})
	}

	BeforeEach(func() {
		fakeBroker = &types.Broker{
			ID:        "brokerID",
			Name:      "brokerName",
			BrokerURL: "http://localhost:8080/broker",
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: "admin",
					Password: "admin",
				},
			},
		}

		reactionError = &v2.HTTPStatusCodeError{
			StatusCode:   http.StatusInternalServerError,
			ErrorMessage: strPtr("error message"),
			Description:  strPtr("response description"),
		}

		brokerID = fakeBroker.ID

		fakeClient = &fake.FakeClient{}
		fakeClientCreateFunc = newFakeClientFunc(fakeClient, nil)
		fakeBrokerStorage = &storagefakes.FakeBroker{}

		logic = BusinessLogic{
			createFunc:    fakeClientCreateFunc,
			brokerStorage: fakeBrokerStorage,
		}

	})

	Describe("GetCatalog", func() {

		var (
			expectedResponse *v2.CatalogResponse
			actualResponse   *v2.CatalogResponse
		)

		callGetCatalog := func(brokerID string) (*v2.CatalogResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/catalog", nil)

			if brokerID != "" {
				pathParams := map[string]string{
					BrokerIDPathParam: brokerID,
				}
				request = mux.SetURLVars(request, pathParams)
			}

			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.GetCatalog(context)
			var resp *v2.CatalogResponse
			if response != nil {
				resp = &response.CatalogResponse
			}
			return resp, err
		}

		BeforeEach(func() {
			actionType = fake.GetCatalog
			expectedResponse = &v2.CatalogResponse{
				Services: []v2.Service{
					{
						Name: "test",
					},
				},
			}
			fakeClient.CatalogReaction = &fake.CatalogReaction{
				Response: expectedResponse,
				Error:    nil,
			}
			fakeBrokerStorage.GetReturns(fakeBroker, nil)

		})

		Context("when no error occurs", func() {
			BeforeEach(func() {
				actualResponse, actualErr = callGetCatalog(brokerID)

				Expect(actualErr).ToNot(HaveOccurred())
			})

			assertAllRelevantInvocationsHappened()

			It("returns proper response", func() {

				Expect(actualResponse).To(Equal(expectedResponse))

			})

		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				brokerID = ""
				actualResponse, actualErr = callGetCatalog(brokerID)
			})

			assertBehaviourWhenBrokerIDPathParameterIsMissing()
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				brokerID = "missingBroker"
				expectedErr = fmt.Errorf("Could not find broker")
				fakeBrokerStorage.GetReturns(nil, storage.ErrNotFound)

				actualResponse, actualErr = callGetCatalog(brokerID)
			})

			assertBehaviourWhenBrokerNotFoundInStorage()
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				expectedErr = reactionError
				logic.createFunc = newFakeClientFunc(fakeClient, expectedErr)

				actualResponse, actualErr = callGetCatalog(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbClientCreation()
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				expectedErr = reactionError

				fakeClient.CatalogReaction = &fake.CatalogReaction{
					Response: nil,
					Error:    expectedErr,
				}

				actualResponse, actualErr = callGetCatalog(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbCall()

		})

	})

	Describe("Provision", func() {

		var (
			expectedResponse *v2.ProvisionResponse
			actualResponse   *v2.ProvisionResponse

			actualRequest *v2.ProvisionRequest
		)

		callProvision := func(brokerID string) (*v2.ProvisionResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/service_instances/{instance_id}", nil)

			if brokerID != "" {
				pathParams := map[string]string{
					BrokerIDPathParam: brokerID,
				}
				request = mux.SetURLVars(request, pathParams)
			}

			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.Provision(actualRequest, context)
			var resp *v2.ProvisionResponse
			if response != nil {
				resp = &response.ProvisionResponse
			}
			return resp, err
		}

		BeforeEach(func() {
			actionType = fake.ProvisionInstance

			actualRequest = &v2.ProvisionRequest{}
			expectedResponse = &v2.ProvisionResponse{
				Async:        false,
				DashboardURL: strPtr("http://localhost:8080"),
				OperationKey: nil,
			}
			fakeClient.ProvisionReaction = &fake.ProvisionReaction{
				Response: expectedResponse,
				Error:    nil,
			}
			fakeBrokerStorage.GetReturns(fakeBroker, nil)

		})

		Context("when no error occurs", func() {
			BeforeEach(func() {
				actualResponse, actualErr = callProvision(brokerID)

				Expect(actualErr).ToNot(HaveOccurred())
			})

			assertAllRelevantInvocationsHappened()

			It("returns proper response", func() {
				Expect(actualResponse).To(Equal(expectedResponse))

			})

		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				brokerID = ""
				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenBrokerIDPathParameterIsMissing()
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				brokerID = "missingBroker"
				expectedErr = fmt.Errorf("Could not find broker")
				fakeBrokerStorage.GetReturns(nil, storage.ErrNotFound)

				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenBrokerNotFoundInStorage()
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				expectedErr = reactionError
				logic.createFunc = newFakeClientFunc(fakeClient, expectedErr)

				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbClientCreation()
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				expectedErr = reactionError

				fakeClient.ProvisionReaction = &fake.ProvisionReaction{
					Response: nil,
					Error:    expectedErr,
				}

				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbCall()

		})

	})

	Describe("Desprovision", func() {

		var (
			expectedResponse *v2.DeprovisionResponse
			actualResponse   *v2.DeprovisionResponse

			actualRequest *v2.DeprovisionRequest
		)

		callDeprovision := func(brokerID string) (*v2.DeprovisionResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/service_instances/{instance_id}", nil)

			if brokerID != "" {
				pathParams := map[string]string{
					BrokerIDPathParam: brokerID,
				}
				request = mux.SetURLVars(request, pathParams)
			}

			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.Deprovision(actualRequest, context)
			var resp *v2.DeprovisionResponse
			if response != nil {
				resp = &response.DeprovisionResponse
			}
			return resp, err
		}

		BeforeEach(func() {
			actionType = fake.DeprovisionInstance

			actualRequest = &v2.DeprovisionRequest{}
			expectedResponse = &v2.DeprovisionResponse{
				Async:        false,
				OperationKey: nil,
			}
			fakeClient.DeprovisionReaction = &fake.DeprovisionReaction{
				Response: expectedResponse,
				Error:    nil,
			}
			fakeBrokerStorage.GetReturns(fakeBroker, nil)

		})

		Context("when no error occurs", func() {
			BeforeEach(func() {
				actualResponse, actualErr = callDeprovision(brokerID)

				Expect(actualErr).ToNot(HaveOccurred())
			})

			assertAllRelevantInvocationsHappened()

			It("returns proper response", func() {
				Expect(actualResponse).To(Equal(expectedResponse))

			})

		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				brokerID = ""
				actualResponse, actualErr = callDeprovision(brokerID)
			})

			assertBehaviourWhenBrokerIDPathParameterIsMissing()
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				brokerID = "missingBroker"
				expectedErr = fmt.Errorf("Could not find broker")
				fakeBrokerStorage.GetReturns(nil, storage.ErrNotFound)

				actualResponse, actualErr = callDeprovision(brokerID)
			})

			assertBehaviourWhenBrokerNotFoundInStorage()
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				expectedErr = reactionError
				logic.createFunc = newFakeClientFunc(fakeClient, expectedErr)

				actualResponse, actualErr = callDeprovision(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbClientCreation()
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				expectedErr = reactionError

				fakeClient.DeprovisionReaction = &fake.DeprovisionReaction{
					Response: nil,
					Error:    expectedErr,
				}

				actualResponse, actualErr = callDeprovision(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbCall()

		})

	})

	Describe("Last Operation", func() {
		var (
			expectedResponse *v2.LastOperationResponse
			actualResponse   *v2.LastOperationResponse

			actualRequest *v2.LastOperationRequest
		)

		callLastOperation := func(brokerID string) (*v2.LastOperationResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/service_instances/{instance_id}/last_operation", nil)

			if brokerID != "" {
				pathParams := map[string]string{
					BrokerIDPathParam: brokerID,
				}
				request = mux.SetURLVars(request, pathParams)
			}

			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.LastOperation(actualRequest, context)
			var resp *v2.LastOperationResponse
			if response != nil {
				resp = &response.LastOperationResponse
			}
			return resp, err
		}

		BeforeEach(func() {
			actionType = fake.PollLastOperation

			actualRequest = &v2.LastOperationRequest{}
			expectedResponse = &v2.LastOperationResponse{
				State:       "pending",
				Description: strPtr("test"),
			}
			fakeClient.PollLastOperationReaction = &fake.PollLastOperationReaction{
				Response: expectedResponse,
				Error:    nil,
			}
			fakeBrokerStorage.GetReturns(fakeBroker, nil)

		})

		Context("when no error occurs", func() {
			BeforeEach(func() {
				actualResponse, actualErr = callLastOperation(brokerID)

				Expect(actualErr).ToNot(HaveOccurred())
			})

			assertAllRelevantInvocationsHappened()

			It("returns proper response", func() {
				Expect(actualResponse).To(Equal(expectedResponse))

			})

		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				brokerID = ""
				actualResponse, actualErr = callLastOperation(brokerID)
			})

			assertBehaviourWhenBrokerIDPathParameterIsMissing()
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				brokerID = "missingBroker"
				expectedErr = fmt.Errorf("Could not find broker")
				fakeBrokerStorage.GetReturns(nil, storage.ErrNotFound)

				actualResponse, actualErr = callLastOperation(brokerID)
			})

			assertBehaviourWhenBrokerNotFoundInStorage()
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				expectedErr = reactionError
				logic.createFunc = newFakeClientFunc(fakeClient, expectedErr)

				actualResponse, actualErr = callLastOperation(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbClientCreation()
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				expectedErr = reactionError

				fakeClient.PollLastOperationReaction = &fake.PollLastOperationReaction{
					Response: nil,
					Error:    expectedErr,
				}

				actualResponse, actualErr = callLastOperation(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbCall()

		})

	})

	Describe("Bind", func() {
		var (
			expectedResponse *v2.BindResponse
			actualResponse   *v2.BindResponse

			actualRequest *v2.BindRequest
		)

		callBind := func(brokerID string) (*v2.BindResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/service_instances/{instance_id}/service_bindings/{binding_id}", nil)

			if brokerID != "" {
				pathParams := map[string]string{
					BrokerIDPathParam: brokerID,
				}
				request = mux.SetURLVars(request, pathParams)
			}

			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.Bind(actualRequest, context)
			var resp *v2.BindResponse
			if response != nil {
				resp = &response.BindResponse
			}
			return resp, err
		}

		BeforeEach(func() {
			actionType = fake.Bind

			actualRequest = &v2.BindRequest{}

			expectedResponse = &v2.BindResponse{
				Async: false,
				Credentials: map[string]interface{}{
					"username": "admin",
					"password": "admin",
				},
			}
			fakeClient.BindReaction = &fake.BindReaction{
				Response: expectedResponse,
				Error:    nil,
			}
			fakeBrokerStorage.GetReturns(fakeBroker, nil)

		})

		Context("when no error occurs", func() {
			BeforeEach(func() {
				actualResponse, actualErr = callBind(brokerID)

				Expect(actualErr).ToNot(HaveOccurred())
			})

			assertAllRelevantInvocationsHappened()

			It("returns proper response", func() {
				Expect(actualResponse).To(Equal(expectedResponse))

			})

		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				brokerID = ""
				actualResponse, actualErr = callBind(brokerID)
			})

			assertBehaviourWhenBrokerIDPathParameterIsMissing()
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				brokerID = "missingBroker"
				expectedErr = fmt.Errorf("Could not find broker")
				fakeBrokerStorage.GetReturns(nil, storage.ErrNotFound)

				actualResponse, actualErr = callBind(brokerID)
			})

			assertBehaviourWhenBrokerNotFoundInStorage()
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				expectedErr = reactionError
				logic.createFunc = newFakeClientFunc(fakeClient, expectedErr)

				actualResponse, actualErr = callBind(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbClientCreation()
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				expectedErr = reactionError

				fakeClient.BindReaction = &fake.BindReaction{
					Response: nil,
					Error:    expectedErr,
				}

				actualResponse, actualErr = callBind(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbCall()

		})

	})

	Describe("Unbind", func() {
		var (
			expectedResponse *v2.UnbindResponse
			actualResponse   *v2.UnbindResponse

			actualRequest *v2.UnbindRequest
		)

		callUnbind := func(brokerID string) (*v2.UnbindResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/service_instances/{instance_id}", nil)

			if brokerID != "" {
				pathParams := map[string]string{
					BrokerIDPathParam: brokerID,
				}
				request = mux.SetURLVars(request, pathParams)
			}

			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.Unbind(actualRequest, context)
			var resp *v2.UnbindResponse
			if response != nil {
				resp = &response.UnbindResponse
			}
			return resp, err
		}

		BeforeEach(func() {
			actionType = fake.Unbind

			actualRequest = &v2.UnbindRequest{}
			expectedResponse = &v2.UnbindResponse{
				Async:        false,
				OperationKey: nil,
			}
			fakeClient.UnbindReaction = &fake.UnbindReaction{
				Response: expectedResponse,
				Error:    nil,
			}
			fakeBrokerStorage.GetReturns(fakeBroker, nil)

		})

		Context("when no error occurs", func() {
			BeforeEach(func() {
				actualResponse, actualErr = callUnbind(brokerID)

				Expect(actualErr).ToNot(HaveOccurred())
			})

			assertAllRelevantInvocationsHappened()

			It("returns proper response", func() {
				Expect(actualResponse).To(Equal(expectedResponse))

			})

		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				brokerID = ""
				actualResponse, actualErr = callUnbind(brokerID)
			})

			assertBehaviourWhenBrokerIDPathParameterIsMissing()
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				brokerID = "missingBroker"
				expectedErr = fmt.Errorf("Could not find broker")
				fakeBrokerStorage.GetReturns(nil, storage.ErrNotFound)

				actualResponse, actualErr = callUnbind(brokerID)
			})

			assertBehaviourWhenBrokerNotFoundInStorage()
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				expectedErr = reactionError
				logic.createFunc = newFakeClientFunc(fakeClient, expectedErr)

				actualResponse, actualErr = callUnbind(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbClientCreation()
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				expectedErr = reactionError

				fakeClient.UnbindReaction = &fake.UnbindReaction{
					Response: nil,
					Error:    expectedErr,
				}

				actualResponse, actualErr = callUnbind(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbCall()

		})

	})

	Describe("Update", func() {
		var (
			expectedResponse *v2.UpdateInstanceResponse
			actualResponse   *v2.UpdateInstanceResponse

			actualRequest *v2.UpdateInstanceRequest
		)

		callProvision := func(brokerID string) (*v2.UpdateInstanceResponse, error) {
			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodGet, "/osb/"+brokerID+"/v2/service_instances/{instance_id}", nil)

			if brokerID != "" {
				pathParams := map[string]string{
					BrokerIDPathParam: brokerID,
				}
				request = mux.SetURLVars(request, pathParams)
			}

			context := &broker.RequestContext{
				Writer:  recorder,
				Request: request,
			}
			response, err := logic.Update(actualRequest, context)
			var resp *v2.UpdateInstanceResponse
			if response != nil {
				resp = &response.UpdateInstanceResponse
			}
			return resp, err
		}

		BeforeEach(func() {
			actionType = fake.UpdateInstance

			actualRequest = &v2.UpdateInstanceRequest{}
			expectedResponse = &v2.UpdateInstanceResponse{
				Async:        false,
				OperationKey: nil,
			}
			fakeClient.UpdateInstanceReaction = &fake.UpdateInstanceReaction{
				Response: expectedResponse,
				Error:    nil,
			}
			fakeBrokerStorage.GetReturns(fakeBroker, nil)

		})

		Context("when no error occurs", func() {
			BeforeEach(func() {
				actualResponse, actualErr = callProvision(brokerID)

				Expect(actualErr).ToNot(HaveOccurred())
			})

			assertAllRelevantInvocationsHappened()

			It("returns proper response", func() {
				Expect(actualResponse).To(Equal(expectedResponse))

			})

		})

		Context("when brokerID path parameter is missing", func() {
			BeforeEach(func() {
				brokerID = ""
				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenBrokerIDPathParameterIsMissing()
		})

		Context("when broker with brokerID is not found in the storage", func() {
			BeforeEach(func() {
				brokerID = "missingBroker"
				expectedErr = fmt.Errorf("Could not find broker")
				fakeBrokerStorage.GetReturns(nil, storage.ErrNotFound)

				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenBrokerNotFoundInStorage()
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				expectedErr = reactionError
				logic.createFunc = newFakeClientFunc(fakeClient, expectedErr)

				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbClientCreation()
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				expectedErr = reactionError

				fakeClient.UpdateInstanceReaction = &fake.UpdateInstanceReaction{
					Response: nil,
					Error:    expectedErr,
				}

				actualResponse, actualErr = callProvision(brokerID)
			})

			assertBehaviourWhenErrorOccursDuringOsbCall()

		})

	})

	Describe("Validate Broker API Version", func() {
		It("doesn't return error", func() {
			err := logic.ValidateBrokerAPIVersion(v2.LatestAPIVersion().HeaderValue())
			Expect(err).To(Not(HaveOccurred()))
		})
	})

})

func strPtr(s string) *string {
	return &s
}
