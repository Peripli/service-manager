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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Peripli/service-manager/test"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/multitenancy"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/tidwall/gjson"

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
	TenantIdentifier            = "tenant"
	TenantValue                 = "tenant_value"
	plan0CatalogID              = "plan0CatalogID"
	plan1CatalogID              = "plan1CatalogID"
	plan2CatalogID              = "plan2CatalogID"
	plan3CatalogID              = "plan3CatalogID"
	service0CatalogID           = "service0CatalogID"
	service1CatalogID           = "service1CatalogID"
	organizationGUID            = "1113aa0-124e-4af2-1526-6bfacf61b111"
	SID                         = "12345"
	timeoutDuration             = time.Millisecond * 500
	additionalDelayAfterTimeout = time.Millisecond * 5
	testTimeout                 = 10

	brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
	brokerAPIVersionHeaderValue = "2.13"
)

var (
	ctx                 *common.TestContext
	SMWithBasicPlatform *common.SMExpect

	brokerServerWithEmptyCatalog *common.BrokerServer
	emptyCatalogBrokerID         string
	smUrlToEmptyCatalogBroker    string

	brokerServerWithSimpleCatalog    *common.BrokerServer
	simpleBrokerCatalogID            string
	smUrlToSimpleBrokerCatalogBroker string

	stoppedBrokerServer  *common.BrokerServer
	stoppedBrokerID      string
	smUrlToStoppedBroker string

	brokerServer *common.BrokerServer
	brokerID     string
	brokerName   string
	smBrokerURL  string

	provisionRequestBody string

	brokerPlatformCredentialsIDMap map[string]brokerPlatformCredentials
	utils                          *common.BrokerUtils
)

type brokerPlatformCredentials struct {
	username string
	password string
}

var _ = BeforeSuite(func() {
	ctx = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
		Expect(set.Set("server.request_timeout", timeoutDuration.String())).ToNot(HaveOccurred())
		Expect(set.Set("httpclient.response_header_timeout", timeoutDuration.String())).ToNot(HaveOccurred())
		Expect(set.Set("httpclient.timeout", timeoutDuration.String())).ToNot(HaveOccurred())
	}).WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
		_, err := smb.EnableMultitenancy(TenantIdentifier, func(request *web.Request) (string, error) {
			extractTenantFromToken := multitenancy.ExtractTenantFromTokenWrapperFunc("zid")
			user, ok := web.UserFromContext(request.Context())
			if !ok {
				return "", nil
			}
			var userData json.RawMessage
			if err := user.Data(&userData); err != nil {
				return "", fmt.Errorf("could not unmarshal claims from token: %s", err)
			}
			clientIDFromToken := gjson.GetBytes([]byte(userData), "cid").String()
			if "tenancyClient" != clientIDFromToken {
				return "", nil
			}
			user.AccessLevel = web.TenantAccess
			request.Request = request.WithContext(web.ContextWithUser(request.Context(), user))
			return extractTenantFromToken(request)
		})
		return err
	}).Build()

	SMWithBasicPlatform = &common.SMExpect{Expect: ctx.SMWithBasic.Expect}

	brokerPlatformCredentialsIDMap = make(map[string]brokerPlatformCredentials)

	butils := ctx.RegisterBrokerWithCatalog(common.NewEmptySBCatalog())
	emptyCatalogBrokerID = butils.Broker.ID
	brokerServerWithEmptyCatalog = butils.Broker.BrokerServer

	smUrlToEmptyCatalogBroker = brokerServerWithEmptyCatalog.URL() + "/v1/osb/" + emptyCatalogBrokerID
	username, password := test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, emptyCatalogBrokerID)
	brokerPlatformCredentialsIDMap[emptyCatalogBrokerID] = brokerPlatformCredentials{
		username: username,
		password: password,
	}

	simpleBrokerCatalogID, _, brokerServerWithSimpleCatalog = ctx.RegisterBrokerWithCatalog(simpleCatalog).GetBrokerAsParams()
	smUrlToSimpleBrokerCatalogBroker = brokerServerWithSimpleCatalog.URL() + "/v1/osb/" + simpleBrokerCatalogID
	common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, simpleBrokerCatalogID)
	username, password = test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, simpleBrokerCatalogID)
	brokerPlatformCredentialsIDMap[simpleBrokerCatalogID] = brokerPlatformCredentials{
		username: username,
		password: password,
	}

	plan0 := common.GenerateTestPlanWithID(plan0CatalogID)
	service0 := common.GenerateTestServiceWithPlansWithID(service0CatalogID, plan0)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(service0)

	stoppedBrokerID, _, stoppedBrokerServer = ctx.RegisterBrokerWithCatalog(catalog).GetBrokerAsParams()
	common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, stoppedBrokerID)
	stoppedBrokerServer.Close()
	smUrlToStoppedBroker = stoppedBrokerServer.URL() + "/v1/osb/" + stoppedBrokerID
	username, password = test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, stoppedBrokerID)
	brokerPlatformCredentialsIDMap[stoppedBrokerID] = brokerPlatformCredentials{
		username: username,
		password: password,
	}

	plan1 := common.GenerateTestPlanWithID(plan1CatalogID)
	plan2 := common.GenerateTestPlanWithID(plan2CatalogID)
	plan3 := common.GenerateTestPlanWithID(plan3CatalogID)

	service1 := common.GenerateTestServiceWithPlansWithID(service1CatalogID, plan1, plan2, plan3)
	catalog = common.NewEmptySBCatalog()
	catalog.AddService(service1)

	var brokerObject common.Object

	utils = ctx.RegisterBrokerWithCatalog(catalog)
	brokerID = utils.Broker.ID
	brokerObject = utils.Broker.JSON
	brokerServer = utils.Broker.BrokerServer
	utils.BrokerWithTLS = ctx.RegisterBrokerWithRandomCatalogAndTLS(ctx.SMWithOAuth).BrokerWithTLS

	plans := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("catalog_id in ('%s','%s')", plan1CatalogID, plan2CatalogID)).Iter()
	for _, p := range plans {
		common.RegisterVisibilityForPlanAndPlatform(ctx.SMWithOAuth, p.Object().Value("id").String().Raw(), ctx.TestPlatform.ID)
	}
	smBrokerURL = brokerServer.URL() + "/v1/osb/" + brokerID
	brokerName = brokerObject["name"].(string)

	username, password = test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, brokerID)
	brokerPlatformCredentialsIDMap[brokerID] = brokerPlatformCredentials{
		username: username,
		password: password,
	}
})

var _ = BeforeEach(func() {
	resetBrokersHandlers()
	resetBrokersCallHistory()
	provisionRequestBody = buildRequestBody(service1CatalogID, plan1CatalogID)

	credentials := brokerPlatformCredentialsIDMap[brokerID]
	ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)
})

var _ = JustAfterEach(func() {
	common.RemoveAllOperations(ctx.SMRepository)
	common.RemoveAllInstances(ctx)
	common.RemoveAllOperations(ctx.SMRepository)
})

var _ = AfterSuite(func() {
	ctx.Cleanup()
})

func assertMissingBrokerError(req *httpexpect.Response) {
	req.Status(http.StatusNotFound).JSON().Object().
		Value("description").String().Contains("could not find") // broker or offering
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

func gzipWrite(w io.Writer, data []byte) error {
	// Write gzipped data to the client
	gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	defer gw.Close()
	gw.Write(data)
	return err
}

func gzipHandler(statusCode int, responseBody string) func(rw http.ResponseWriter, _ *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Accept-Encoding") == "gzip" {
			rw.Header().Set("Content-Encoding", "gzip")
			rw.WriteHeader(statusCode)
			var buf bytes.Buffer
			gzipWrite(&buf, []byte(responseBody))
			rw.Write(buf.Bytes())
		} else {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(statusCode)
			rw.Write([]byte(responseBody))
		}
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
			"organization_guid": "%s",
			"organization_name": "system",
			"space_guid": "aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
			"space_name": "development",
			"instance_name": "my-db",
			"%s":"%s"
		},
		"maintenance_info": {
			"version": "old"
		}
}`, serviceID, planID, organizationGUID, TenantIdentifier, TenantValue)
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
			"organization_guid": "%s",
			"organization_name": "system",
			"space_guid": "aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
			"space_name": "development",
			"instance_name": "my-db",
			"%s":"%s"
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
}`, serviceID, newPlanID, organizationGUID, TenantIdentifier, TenantValue, serviceID, oldPlanID)
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
	ResourceType types.ObjectType
	Errors       json.RawMessage
	ExternalID   string
}

func verifyOperationExists(operationExpectations operationExpectations) {
	byResourceID := query.ByField(query.EqualsOperator, "resource_id", operationExpectations.ResourceID)
	byType := query.ByField(query.EqualsOperator, "type", string(operationExpectations.Type))
	orderByCreation := query.OrderResultBy("paging_sequence", query.DescOrder)

	objectList, err := ctx.SMRepository.List(context.TODO(), types.OperationType, byType, byResourceID, orderByCreation)
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
	orderByCreation := query.OrderResultBy("paging_sequence", query.DescOrder)
	criterias := append([]query.Criterion{}, byResourceID, orderByCreation)
	if len(operationTypes) != 0 {
		byOperationTypes := query.ByField(query.InOperator, "type", fmt.Sprintf("(%s)", strings.Join(operationTypes, ",")))
		criterias = append(criterias, byOperationTypes)
	}
	objectList, err := ctx.SMRepository.List(context.TODO(), types.OperationType, criterias...)
	Expect(err).ToNot(HaveOccurred())
	Expect(objectList.Len()).To(BeZero())
}
