package provision_timeout

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/tidwall/sjson"
	"net/http"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

// TestProvisionTimeout tests the context timeout filter for provision requests
func TestProvisionTimeout(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provision Timeout Tests Suite")
}

const (
	TenantIdentifier  = "tenant"
	TenantValue       = "tenant_value"
	plan1CatalogID    = "plan1CatalogID"
	service1CatalogID = "service1CatalogID"
	organizationGUID  = "1113aa0-124e-4af2-1526-6bfacf61b111"
	SID               = "12345"
	SID2              = "123456"
	timeoutDuration   = time.Millisecond * 1500
	testTimeout       = 10

	brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
	brokerAPIVersionHeaderValue = "2.16"
)

var (
	ctx                 *common.TestContext
	SMWithBasicPlatform *common.SMExpect

	brokerServer *common.BrokerServer
	brokerID     string
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
		Expect(set.Set("server.provision_timeout", "0.1s")).ToNot(HaveOccurred())
		Expect(set.Set("server.request_timeout", "4s")).ToNot(HaveOccurred())
		Expect(set.Set("httpclient.response_header_timeout", timeoutDuration.String())).ToNot(HaveOccurred())
		Expect(set.Set("httpclient.timeout", timeoutDuration.String())).ToNot(HaveOccurred())
	}).WithTenantTokenClaims(map[string]interface{}{
		"cid": "tenancyClient",
		"zid": TenantValue,
	}).Build()

	SMWithBasicPlatform = &common.SMExpect{Expect: ctx.SMWithBasic.Expect}

	brokerPlatformCredentialsIDMap = make(map[string]brokerPlatformCredentials)

	plan1 := common.GenerateTestPlanWithID(plan1CatalogID)

	service1 := common.GenerateTestServiceWithPlansWithID(service1CatalogID, plan1)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(service1)

	utils = ctx.RegisterBrokerWithCatalog(catalog)
	brokerID = utils.Broker.ID
	brokerServer = utils.Broker.BrokerServer

	plans := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("catalog_id in ('%s')", plan1CatalogID)).Iter()
	for _, p := range plans {
		common.RegisterVisibilityForPlanAndPlatform(ctx.SMWithOAuth, p.Object().Value("id").String().Raw(), ctx.TestPlatform.ID)
	}
	smBrokerURL = ctx.Servers[common.SMServer].URL() + "/v1/osb/" + brokerID

	username, password := test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, brokerID)
	brokerPlatformCredentialsIDMap[brokerID] = brokerPlatformCredentials{
		username: username,
		password: password,
	}

	provisionRequestBody = buildRequestBody(service1CatalogID, plan1CatalogID)
})

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

func slowResponseHandler(seconds int, done chan struct{}) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		if f, ok := rw.(http.Flusher); ok {
			for i := 1; i <= seconds*10; i++ {
				_, err := fmt.Fprintf(rw, "Chunk #%d\n", i)
				if err != nil {
					break
				}
				f.Flush()
				time.Sleep(100 * time.Millisecond)
			}
		}
		<-done
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

func verifyOperationDoesNotExist(resourceID string, operationTypes ...string) {
	byResourceID := query.ByField(query.EqualsOperator, "resource_id", resourceID)
	orderByCreation := query.OrderResultBy("paging_sequence", query.DescOrder)
	criterias := append([]query.Criterion{}, byResourceID, orderByCreation)
	if len(operationTypes) != 0 {
		byOperationTypes := query.ByField(query.InOperator, "type", operationTypes...)
		criterias = append(criterias, byOperationTypes)
	}
	objectList, err := ctx.SMRepository.List(context.TODO(), types.OperationType, criterias...)
	Expect(err).ToNot(HaveOccurred())
	Expect(objectList.Len()).To(BeZero())
}

func parameterizedHandler(statusCode int, responseBody string) func(rw http.ResponseWriter, _ *http.Request) {
	return func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(statusCode)
		rw.Write([]byte(responseBody))
	}
}
