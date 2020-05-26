package cascade_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/multitenancy"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	. "github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	"net/http"
	"time"
)

var (
	ctx                   *TestContext
	brokerServer          *BrokerServer
	brokerID              string
	plan                  types.Object
	platformID            string
	tenantOperationsCount = 11 //the number of operations that will be created after tenant creation in JustBeforeEach
	rootOpID              = "op1"
	tenantID              = "tenant_value"
	osbInstanceID        = "test-instance"
)

const (
	polling                 = 1 * time.Millisecond
	maintainerRetry         = 1 * time.Second
	actionTimeout           = 2 * time.Second
	cleanupInterval         = 9999 * time.Hour
	pollCascade             = 500 * time.Millisecond
	reconciliation          = 9999 * time.Hour
	lifespan                = 1 * time.Millisecond
	cascadeOrphanMitigation = 5 * time.Second
)

var queryForOperationsInTheSameTree = query.ByField(query.EqualsOperator, "cascade_root_id", rootOpID)
var queryForRoot = query.ByField(query.EqualsOperator, "id", rootOpID)

var queryForInstanceOperations = query.ByField(query.EqualsOperator, "resource_type", types.ServiceInstanceType.String())
var queryForBindingsOperations = query.ByField(query.EqualsOperator, "resource_type", types.ServiceBindingType.String())
var queryForOrphanMitigationOperations = query.ByField(query.NotEqualsOperator, "deletion_scheduled", operations.ZeroTime)

//var queryForPlatformOperations := query.ByField(query.EqualsOperator, "resource_type", types.PlatformType.String())
//var queryForTenantOperations := query.ByField(query.EqualsOperator, "resource_type", types.TenantType.String())
//var queryForBrokerOperations := query.ByField(query.EqualsOperator, "resource_type", types.ServiceBrokerType.String())
var querySucceeded = query.ByField(query.EqualsOperator, "state", string(types.SUCCEEDED))

//var queryInProgress := query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS))
//var queryPending := query.ByField(query.EqualsOperator, "state", string(types.PENDING))
var queryFailedOperations = query.ByField(query.EqualsOperator, "state", string(types.FAILED))

//var queryForDuplications := query.ByField(query.EqualsOrNilOperator, "external_id", "")

var _ = BeforeEach(func() {
	postHookWithOperationsConfig := func() func(e env.Environment, servers map[string]FakeServer) {
		return func(e env.Environment, servers map[string]FakeServer) {
			e.Set("operations.action_timeout", actionTimeout)
			e.Set("operations.maintainer_retry_interval", maintainerRetry)
			e.Set("operations.polling_interval", polling)
			e.Set("operations.cleanup_interval", cleanupInterval)
			e.Set("operations.poll_cascade_interval", pollCascade)
			e.Set("operations.lifespan", lifespan)
			e.Set("operations.reconciliation_operation_timeout", reconciliation)
			e.Set("operations.cascade_orphan_mitigation_timeout", cascadeOrphanMitigation)
		}
	}
	postHook := postHookWithOperationsConfig()
	ctx = NewTestContextBuilderWithSecurity().WithEnvPostExtensions(postHook).
		WithEnvPreExtensions(func(set *pflag.FlagSet) {
			Expect(set.Set("server.request_timeout", "4s")).ToNot(HaveOccurred())
			Expect(set.Set("httpclient.response_header_timeout", (time.Millisecond * 1500).String())).ToNot(HaveOccurred())
			Expect(set.Set("httpclient.timeout", (time.Millisecond * 1500).String())).ToNot(HaveOccurred())
		}).
		WithTenantTokenClaims(map[string]interface{}{
			"cid": "tenancyClient",
			"zid": tenantID,
		}).
		WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			_, err := smb.EnableMultitenancy("tenant", func(request *web.Request) (string, error) {
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
		}).
		Build()
})

var _ = JustBeforeEach(func() {
	brokerID, brokerServer = registerSubaccountScopedBroker(ctx, "test-service", "plan-service")
	platformID = registerSubaccountScopedPlatform(ctx, "platform1")
	var err error
	plan, err = ctx.SMRepository.Get(context.Background(), types.ServicePlanType, query.ByField(query.EqualsOperator, "catalog_id", "plan-service"))
	Expect(err).NotTo(HaveOccurred())
	createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
		"name":            "test-instance-smaap",
		"service_plan_id": plan.GetID(),
	})
	createOSBInstance(ctx, ctx.SMWithBasic, brokerID, osbInstanceID, map[string]interface{}{
		"service_id":        "test-service",
		"plan_id":           "plan-service",
		"organization_guid": "my-org",
	})
	createOSBBinding(ctx, ctx.SMWithBasic, brokerID, osbInstanceID, "binding1", map[string]interface{}{
		"service_id":        "test-service",
		"plan_id":           "plan-service",
		"organization_guid": "my-org",
	})
	createOSBBinding(ctx, ctx.SMWithBasic, brokerID, osbInstanceID, "binding2", map[string]interface{}{
		"service_id":        "test-service",
		"plan_id":           "plan-service",
		"organization_guid": "my-org",
	})
})

func createOSBInstance(ctx *TestContext, sm *SMExpect, brokerID string, instanceID string, osbContext map[string]interface{}) {
	smBrokerURL := ctx.Servers[SMServer].URL() + "/v1/osb/" + brokerID
	sm.PUT(smBrokerURL+"/v2/service_instances/"+instanceID).
		WithHeader("X-Broker-API-Version", "2.13").
		WithJSON(osbContext).
		WithQuery("accepts_incomplete", false).
		Expect().
		Status(http.StatusCreated)
}

func createSMAAPInstance(ctx *TestContext, sm *SMExpect, context map[string]interface{}) string {
	smBrokerURL := ctx.Servers[SMServer].URL()
	return sm.POST(smBrokerURL+web.ServiceInstancesURL).
		WithQuery("async", false).
		WithJSON(context).
		Expect().
		Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
}

func createSMAAPBinding(ctx *TestContext, sm *SMExpect, context map[string]interface{}) {
	smBrokerURL := ctx.Servers[SMServer].URL()
	sm.POST(smBrokerURL+web.ServiceBindingsURL).
		WithQuery("async", false).
		WithJSON(context).
		Expect().
		Status(http.StatusCreated)
}

func createOSBBinding(ctx *TestContext, sm *SMExpect, brokerID string, instanceID string, bindingID string, osbContext map[string]interface{}) {
	smBrokerURL := ctx.Servers[SMServer].URL() + "/v1/osb/" + brokerID
	sm.PUT(smBrokerURL+"/v2/service_instances/"+instanceID+"/service_bindings/"+bindingID).
		WithHeader("X-Broker-API-Version", "2.13").
		WithJSON(osbContext).
		WithQuery("accepts_incomplete", false).
		Expect().
		Status(http.StatusCreated)
}

func registerSubaccountScopedBroker(ctx *TestContext, serviceNameID string, planID string) (string, *BrokerServer) {
	// registering a tenant scope broker
	catalog := SimpleCatalog(serviceNameID, planID)
	id, _, brokerServer := ctx.RegisterBrokerWithCatalogAndLabelsExpect(catalog, map[string]interface{}{}, ctx.SMWithOAuthForTenant).GetBrokerAsParams()
	brokerServer.ShouldRecordRequests(false)

	brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut, func(req *http.Request) (int, map[string]interface{}) {
		return http.StatusCreated, Object{}
	})
	brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut, func(req *http.Request) (int, map[string]interface{}) {
		return http.StatusCreated, Object{}
	})
	brokerServer.ServiceInstanceHandlerFunc(http.MethodPost, http.MethodPost, func(req *http.Request) (int, map[string]interface{}) {
		return http.StatusCreated, Object{}
	})

	brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
		return http.StatusAccepted, Object{"async": true}
	})
	brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
		return http.StatusAccepted, Object{"async": true}
	})

	registerInstanceLastOPHandlers(brokerServer, types.SUCCEEDED)
	registerBindingLastOPHandlers(brokerServer, types.SUCCEEDED)
	CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, id)
	return id, brokerServer
}

func registerInstanceLastOPHandlers(brokerServer *BrokerServer, state types.OperationState) {
	brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
		return http.StatusOK, Object{"state": state}
	})
}

func registerBindingLastOPHandlers(brokerServer *BrokerServer, state types.OperationState) {
	brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
		return http.StatusOK, Object{"state": state}
	})
}

func registerSubaccountScopedPlatform(ctx *TestContext, name string) string {
	platform := MakePlatform(name, name, "cf", "descr")
	reply := ctx.SMWithOAuthForTenant.POST(web.PlatformsURL).
		WithJSON(platform).
		Expect().
		Status(http.StatusCreated).
		JSON().
		Object()

	id := reply.Value("id").String().NotEmpty().Raw()
	platform["id"] = id
	MapContains(reply.Raw(), platform)
	basic := reply.Value("credentials").Object().Value("basic").Object()
	username := basic.Value("username").String().NotEmpty()
	secret := basic.Value("password").String().NotEmpty()

	// creating a tenant instance in tenant platform
	ctx.SMWithBasic.SetBasicCredentials(ctx, username.Raw(), secret.Raw())
	return id
}

func SimpleCatalog(serviceID, planID string) SBCatalog {
	return SBCatalog(fmt.Sprintf(`{
	  "services": [{
			"bindings_retrievable": true,
			"instances_retrievable": true,
			"name": "no-tags-no-metadata",
			"id": "%s",
			"description": "A fake service.",
			"plans": [{
				"bindable": true,
				"name": "fake-plan-1",
				"id": "%s",
				"description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections."
			}]
		}]
	}`, serviceID, planID))
}

func fetchFullTree(repository storage.TransactionalRepository, rootID string) (*tree, error) {
	fullTree := tree{
		byResourceID:  make(map[string][]*types.Operation),
		byParentID:    make(map[string][]*types.Operation),
		byOperationID: make(map[string]*types.Operation),
	}

	operations, err := repository.List(context.Background(), types.OperationType, query.ByField(query.EqualsOperator, "cascade_root_id", rootID))
	if err != nil {
		return nil, err
	}
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		if operation.ParentID == "" {
			fullTree.root = operation
			fullTree.byParentID[""] = append(fullTree.byParentID[""], operation)
		} else {
			fullTree.byParentID[operation.ParentID] = append(fullTree.byParentID[operation.ParentID], operation)
		}
		fullTree.byResourceID[operation.ResourceID] = append(fullTree.byResourceID[operation.ResourceID], operation)
		fullTree.byOperationID[operation.ID] = operation
	}
	return &fullTree, nil
}

func AssertOperationCount(expect func(count int), criterion ...query.Criterion) {
	count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, criterion...)
	Expect(err).NotTo(HaveOccurred())
	expect(count)
}

type tree struct {
	root          *types.Operation
	byResourceID  map[string][]*types.Operation
	byParentID    map[string][]*types.Operation
	byOperationID map[string]*types.Operation
}
