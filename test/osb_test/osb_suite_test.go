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

package osb_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	"github.com/gofrs/uuid"
	"github.com/spf13/pflag"
	"github.com/tidwall/sjson"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestOSB tests for OSB API
func TestOSB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OSB API Tests Suite")
}

const (
	plan1CatalogID              = "plan1CatalogID"
	plan2CatalogID              = "plan2CatalogID"
	service1CatalogID           = "service1CatalogID"
	SID                         = "12345"
	timeoutDuration             = time.Millisecond * 500
	additionalDelayAfterTimeout = time.Second

	brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
	brokerAPIVersionHeaderValue = "2.13"
)

var (
	ctx *common.TestContext

	brokerServerWithEmptyCatalog *common.BrokerServer
	emptyCatalogBrokerID         string
	smUrlToEmptyCatalogBroker    string

	smUrlToSimpleBrokerCatalogBroker string

	stoppedBrokerServer  *common.BrokerServer
	stoppedBrokerID      string
	smUrlToStoppedBroker string

	brokerServer *common.BrokerServer
	brokerID     string
	brokerName   string
	smBrokerURL  string

	provisionRequestBody string
)

var _ = BeforeSuite(func() {
	ctx = common.NewTestContextBuilder().WithEnvPreExtensions(func(set *pflag.FlagSet) {
		Expect(set.Set("httpclient.response_header_timeout", timeoutDuration.String())).ToNot(HaveOccurred())
	}).Build()

	emptyCatalogBrokerID, _, brokerServerWithEmptyCatalog = ctx.RegisterBrokerWithCatalog(common.NewEmptySBCatalog())
	smUrlToEmptyCatalogBroker = brokerServerWithEmptyCatalog.URL() + "/v1/osb/" + emptyCatalogBrokerID

	simpleBrokerCatalogID, _, brokerServerWithSimpleCatalog := ctx.RegisterBrokerWithCatalog(simpleCatalog)
	smUrlToSimpleBrokerCatalogBroker = brokerServerWithSimpleCatalog.URL() + "/v1/osb/" + simpleBrokerCatalogID
	common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, simpleBrokerCatalogID)

	stoppedBrokerID, _, stoppedBrokerServer = ctx.RegisterBroker()
	stoppedBrokerServer.Close()
	smUrlToStoppedBroker = stoppedBrokerServer.URL() + "/v1/osb/" + stoppedBrokerID

	plan1 := common.GenerateTestPlanWithID(plan1CatalogID)
	plan2 := common.GenerateTestPlanWithID(plan2CatalogID)

	service1 := common.GenerateTestServiceWithPlansWithID(service1CatalogID, plan1, plan2)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(service1)

	var brokerObject common.Object
	brokerID, brokerObject, brokerServer = ctx.RegisterBrokerWithCatalog(catalog)
	smBrokerURL = brokerServer.URL() + "/v1/osb/" + brokerID
	brokerName = brokerObject["name"].(string)
})

var _ = BeforeEach(func() {
	resetBrokersHandlers()
	resetBrokersCallHistory()
	provisionRequestBody = buildRequestBody(service1CatalogID, plan1CatalogID)
})

var _ = AfterEach(func() {
	common.RemoveAllOperations(ctx.SMRepository)
	common.RemoveAllInstances(ctx.SMRepository)
})

var _ = AfterSuite(func() {
	ctx.Cleanup()
})

func assertMissingBrokerError(req *httpexpect.Response) {
	req.Status(http.StatusNotFound).JSON().Object().
		Value("description").String().Contains("could not find such broker")
}

func assertUnresponsiveBrokerError(req *httpexpect.Response) {
	req.Status(http.StatusBadGateway).JSON().Object().
		Value("description").String().Contains("could not reach service broker")
}

func assertFailingBrokerError(req *httpexpect.Response, expectedStatus int, expectedError string) {
	req.Status(expectedStatus).JSON().Object().
		Value("description").String().Contains(expectedError)
}

func generateRandomQueryParam() (string, string) {
	key, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	value, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	return key.String(), value.String()
}

func findSMPlanIDForCatalogPlanID(catalogPlanID string) string {
	plans := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_id eq '%s'", catalogPlanID))
	plans.Length().Equal(1)
	return plans.First().Object().Value("id").String().Raw()
}

func parameterizedHandler(statusCode int, responseBody string) func(rw http.ResponseWriter, _ *http.Request) {
	return func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(statusCode)
		rw.Write([]byte(responseBody))
	}
}

func queryParameterVerificationHandler(key, value string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		defer GinkgoRecover()
		var status int
		query := request.URL.Query()
		actualValue := query.Get(key)
		Expect(actualValue).To(Equal(value))
		if request.Method == http.MethodPut {
			status = http.StatusCreated
		} else {
			status = http.StatusOK
		}
		common.SetResponse(writer, status, common.Object{})
	}
}

func delayingHandler(done chan<- interface{}) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		brokerDelay := timeoutDuration + additionalDelayAfterTimeout
		timeoutContext, _ := context.WithTimeout(req.Context(), brokerDelay)
		<-timeoutContext.Done()
		common.SetResponse(rw, http.StatusTeapot, common.Object{})
		close(done)
	}
}

func resetBrokersHandlers() {
	brokerServerWithEmptyCatalog.ResetHandlers()
	stoppedBrokerServer.ResetHandlers()
	brokerServer.ResetHandlers()

}

func resetBrokersCallHistory() {
	brokerServerWithEmptyCatalog.ResetCallHistory()
	stoppedBrokerServer.ResetCallHistory()
	brokerServer.ResetCallHistory()
}

func buildRequestBody(serviceID, planID string) string {
	result := fmt.Sprintf(`{
		"service_id":        "%s",
		"plan_id":           "%s",
		"organization_guid": "113aa0-124e-4af2-1526-6bfacf61b111",
		"space_guid":        "aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
		"parameters": {
			"param1": "value1",
			"param2": "value2"
		},
		"context": {
			"platform": "cloudfoundry",
			"organization_guid": "1113aa0-124e-4af2-1526-6bfacf61b111",
			"organization_name": "system",
			"space_guid": "aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
			"space_name": "development",
			"instance_name": "my-db"
		},
		"maintenance_info": {
			"version": "old"
		}
}`, serviceID, planID)
	return result
}
func provisionRequestBodyMapWith(key, value string, idsToRemove ...string) func() map[string]interface{} {
	return func() map[string]interface{} {
		defer GinkgoRecover()
		var err error
		provisionRequestBody, err = sjson.Set(provisionRequestBody, key, value)
		if err != nil {
			Fail(err.Error())
		}
		for _, id := range idsToRemove {
			provisionRequestBody, err = sjson.Delete(provisionRequestBody, id)
			if err != nil {
				Fail(err.Error())
			}
		}
		return common.JSONToMap(provisionRequestBody)
	}
}

func provisionRequestBodyMap(idsToRemove ...string) func() map[string]interface{} {
	return func() map[string]interface{} {

		defer GinkgoRecover()
		var err error
		for _, id := range idsToRemove {
			provisionRequestBody, err = sjson.Delete(provisionRequestBody, id)
			if err != nil {
				Fail(err.Error())
			}
		}
		return common.JSONToMap(provisionRequestBody)
	}
}

func updateRequestBody(serviceID, oldPlanID, newPlanID string) string {
	body := fmt.Sprintf(`{
		"service_id":        "%s",
		"plan_id":           "%s",
		"organization_id": "113aa0-124e-4af2-1526-6bfacf61b111",
		"space_id":        "aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
		"parameters": {
			"param1": "value1",
			"param2": "value2"
		},
		"context": {
			"platform": "cloudfoundry",
			"organization_guid": "1113aa0-124e-4af2-1526-6bfacf61b111",
			"organization_name": "system",
			"space_guid": "aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
			"space_name": "development",
			"instance_name": "my-db"
		},
		"maintenance_info": {
			"version": "new"
		},
		"previous_values": {
			"service_id":        "%s",
			"plan_id":           "%s",
			"organization_id": "113aa0-124e-4af2-1526-6bfacf61b111",
			"space_id":        "aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
			"maintenance_info": {
				"version": "old"
			}
		}
}`, serviceID, newPlanID, serviceID, oldPlanID)
	return body
}

func updateRequestBodyMapWith(key, value string) func() map[string]interface{} {
	return func() map[string]interface{} {
		defer GinkgoRecover()
		var err error
		body := updateRequestBody(service1CatalogID, plan1CatalogID, plan2CatalogID)
		body, err = sjson.Set(body, key, value)
		if err != nil {
			Fail(err.Error())
		}
		return common.JSONToMap(body)
	}
}

func updateRequestBodyMap(idsToRemove ...string) func() map[string]interface{} {
	return func() map[string]interface{} {
		var err error
		body := updateRequestBody(service1CatalogID, plan1CatalogID, plan2CatalogID)
		for _, id := range idsToRemove {
			body, err = sjson.Delete(body, id)
			if err != nil {
				Fail(err.Error())
			}
		}
		return common.JSONToMap(body)
	}
}

type operationExpectations struct {
	Type         types.OperationCategory
	State        types.OperationState
	ResourceID   string
	ResourceType string
	Errors       json.RawMessage
	ExternalID   string
}

func verifyOperationExists(operationExpectations operationExpectations) {
	byResourceID := query.ByField(query.EqualsOperator, "resource_id", operationExpectations.ResourceID)
	byType := query.ByField(query.EqualsOperator, "type", string(operationExpectations.Type))
	orderByCreation := query.OrderResultBy("created_at", query.AscOrder)
	limitToOne := query.LimitResultBy(1)

	objectList, err := ctx.SMRepository.List(context.TODO(), types.OperationType, byType, byResourceID, orderByCreation, limitToOne)
	Expect(err).ToNot(HaveOccurred())
	operation := objectList.ItemAt(0).(*types.Operation)
	Expect(operation.Type).To(Equal(operationExpectations.Type))
	Expect(operation.State).To(Equal(operationExpectations.State))
	Expect(operation.ResourceType).To(Equal(operationExpectations.ResourceType))
	Expect(operation.ResourceID).To(Equal(operationExpectations.ResourceID))
	Expect(operation.ExternalID).To(Equal(operationExpectations.ExternalID))
	Expect(string(operation.Errors)).To(ContainSubstring(string(operationExpectations.Errors)))
}

func verifyOperationDoesNotExist(resourceID string, operationTypes ...string) {
	byResourceID := query.ByField(query.EqualsOperator, "resource_id", resourceID)
	orderByCreation := query.OrderResultBy("created_at", query.AscOrder)
	criterias := append([]query.Criterion{}, byResourceID, orderByCreation)
	if len(operationTypes) != 0 {
		byOperationTypes := query.ByField(query.InOperator, "type", fmt.Sprintf("(%s)", strings.Join(operationTypes, ",")))
		criterias = append(criterias, byOperationTypes)
	}
	objectList, err := ctx.SMRepository.List(context.TODO(), types.OperationType, criterias...)
	Expect(err).ToNot(HaveOccurred())
	Expect(objectList.Len()).To(BeZero())
}
