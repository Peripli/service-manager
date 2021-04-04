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

// TestOSB tests for OSB API
func TestProvisionTimeout(t *testing.T) {
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
	timeoutDuration             = time.Millisecond * 1500
	additionalDelayAfterTimeout = time.Millisecond * 5
	testTimeout                 = 10

	brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
	brokerAPIVersionHeaderValue = "2.16"
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
	shouldStoreBinding             bool
	shouldSaveOperationInContext   bool
	fakeStateResponseBody          []byte
)

const serviceCatalogID = "acb56d7c-XXXX-XXXX-XXXX-feb140a59a67"

var simpleCatalog = fmt.Sprintf(`
{
  "services": [{
		"name": "no-tags-no-metadata",
		"id": "%s",
		"description": "A fake service.",
		"dashboard_client": {
			"id": "id",
			"secret": "secret",
			"redirect_uri": "redirect_uri"		
		},    
		"plans": [{
			"random_extension": "random_extension",
			"name": "fake-plan-1",
			"id": "d3031751-XXXX-XXXX-XXXX-a42377d33202",
			"description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections.",
			"free": false
		}]
	}]
}
`, serviceCatalogID)

type brokerPlatformCredentials struct {
	username string
	password string
}

var _ = BeforeSuite(func() {
	ctx = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
		Expect(set.Set("server.provision_timeout", "2s")).ToNot(HaveOccurred())
		Expect(set.Set("server.request_timeout", "4s")).ToNot(HaveOccurred())
		Expect(set.Set("httpclient.response_header_timeout", timeoutDuration.String())).ToNot(HaveOccurred())
		Expect(set.Set("httpclient.timeout", timeoutDuration.String())).ToNot(HaveOccurred())
	}).WithTenantTokenClaims(map[string]interface{}{
		"cid": "tenancyClient",
		"zid": TenantValue,
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

	simpleBrokerCatalogID, _, brokerServerWithSimpleCatalog = ctx.RegisterBrokerWithCatalog(common.SBCatalog(simpleCatalog)).GetBrokerAsParams()
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
	smBrokerURL = ctx.Servers[common.SMServer].URL() + "/v1/osb/" + brokerID
	brokerName = brokerObject["name"].(string)

	username, password = test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, brokerID)
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
