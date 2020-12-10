package cascade_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	. "github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCascade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cascade Suite Test")
}

var (
	ctx                   *TestContext
	tenantBrokerServer    *BrokerServer
	globalBrokerServer    *BrokerServer
	tenantBrokerID        string
	globalBrokerID        string
	plan                  types.Object
	globalPlan            types.Object
	platformID            string
	tenantOperationsCount = 14 //the number of operations that will be created after tenant creation in JustBeforeEach
	tenantID              = "tenant_value"
	osbInstanceID         = "test-instance"
	osbBindingID          = "test-binding"
	smaapInstanceID1      = ""
	smaapInstanceID2      = ""
	testFullTree          *tree
	printTree             = false

	queryForOperationsInTheSameTree    func(rootID string) query.Criterion
	queryForRoot                       func(rootID string) query.Criterion
	queryForInstanceOperations         query.Criterion
	queryForBindingsOperations         query.Criterion
	queryForOrphanMitigationOperations query.Criterion
	querySucceeded                     query.Criterion
	queryFailures                      query.Criterion

	subaccountResources = make(map[types.ObjectType]int)
)

const (
	polling                 = 1 * time.Millisecond
	maintainerRetry         = 1 * time.Second
	lifespan                = 1 * time.Millisecond
	cascadeOrphanMitigation = 5 * time.Second
	cleanupInterval         = 9999 * time.Hour
	reconciliation          = 9999 * time.Hour
	actionTimeout           = 2 * time.Second
	pollCascade             = 500 * time.Millisecond
)

var _ = AfterEach(func() {
	if printTree && testFullTree != nil {
		fmt.Println()
		fmt.Println(fmt.Sprintf("test (%s) tree: ", CurrentGinkgoTestDescription().FullTestText))
		printFullTree(testFullTree.byParentID, testFullTree.root, 0)
	}
	ctx.CleanupAdditionalResources()
})

var _ = BeforeSuite(func() {
	queryForOperationsInTheSameTree = func(rootID string) query.Criterion {
		return query.ByField(query.EqualsOperator, "cascade_root_id", rootID)
	}
	queryForRoot = func(rootID string) query.Criterion {
		return query.ByField(query.EqualsOperator, "id", rootID)
	}
	queryForInstanceOperations = query.ByField(query.EqualsOperator, "resource_type", types.ServiceInstanceType.String())
	queryForBindingsOperations = query.ByField(query.EqualsOperator, "resource_type", types.ServiceBindingType.String())
	queryForOrphanMitigationOperations = query.ByField(query.NotEqualsOperator, "deletion_scheduled", operations.ZeroTime)
	querySucceeded = query.ByField(query.EqualsOperator, "state", string(types.SUCCEEDED))
	queryFailures = query.ByField(query.EqualsOperator, "state", string(types.FAILED))

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
			_, err := smb.EnableMultitenancy("tenant", ExtractTenantFunc)
			return err
		}).
		Build()
})

func registerGlobalBroker(ctx *TestContext, serviceNameID string, planID string) (string, *BrokerServer) {
	catalog := SimpleCatalog(serviceNameID, planID, generateID())
	id, _, brokerServer := ctx.RegisterBrokerWithCatalogAndLabelsExpect(catalog, map[string]interface{}{}, ctx.SMWithOAuth).GetBrokerAsParams()
	CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, id)
	return id, brokerServer
}

func initTenantResources(createInstances bool) {
	subaccountResources[types.PlatformType]++
	globalBrokerID, globalBrokerServer = registerGlobalBroker(ctx, "global-service", "global-plan")
	tenantBrokerID, tenantBrokerServer = registertenantScopedBroker(ctx, "test-service", "plan-service")

	subaccountResources[types.ServiceBrokerType]++
	var tenantPlatformUser, tenantPlatformSecret string
	platformID, tenantPlatformUser, tenantPlatformSecret = registertenantScopedPlatform(ctx, "platform1")

	var err error
	plan, err = ctx.SMRepository.Get(context.Background(), types.ServicePlanType, query.ByField(query.EqualsOperator, "catalog_id", "plan-service"))
	Expect(err).NotTo(HaveOccurred())
	globalPlan, err = ctx.SMRepository.Get(context.Background(), types.ServicePlanType, query.ByField(query.EqualsOperator, "catalog_id", "global-plan"))
	Expect(err).NotTo(HaveOccurred())

	if createInstances {
		ctx.SMWithBasic.SetBasicCredentials(ctx, ctx.TestPlatform.Credentials.Basic.Username, ctx.TestPlatform.Credentials.Basic.Password)
		// global platform + global broker (tenant child)
		createOSBInstance(ctx, ctx.SMWithBasic, globalBrokerID, "global_platform_global_broker", map[string]interface{}{
			"service_id":        "global-service",
			"plan_id":           "global-plan",
			"organization_guid": "my-orgafsf",
			"context": map[string]string{
				"tenant": tenantID,
			},
		})
		// global platform + tenant scoped broker (broker child)
		createOSBInstance(ctx, ctx.SMWithBasic, tenantBrokerID, "global_platform_tenant_broker", map[string]interface{}{
			"service_id":        "test-service",
			"plan_id":           "plan-service",
			"organization_guid": "my-org",
			"context": map[string]string{
				"tenant": tenantID,
			},
		})

		// SMPlatform + tenant scoped broker (broker child)
		smaapInstanceID1 = createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
			"name":            "test-instance-smaap",
			"service_plan_id": plan.GetID(),
		})
		// SMPlatform + global broker (tenant child)
		smaapInstanceID2 = createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
			"name":            "global-instance-smaap",
			"service_plan_id": globalPlan.GetID(),
		})

		ctx.SMWithBasic.SetBasicCredentials(ctx, tenantPlatformUser, tenantPlatformSecret)
		// tenant scoped platform + global broker (platform child)
		createOSBInstance(ctx, ctx.SMWithBasic, globalBrokerID, "tenant_platform_global_broker", map[string]interface{}{
			"service_id":        "global-service",
			"plan_id":           "global-plan",
			"organization_guid": "my-orgafsf",
			"context": map[string]string{
				"tenant": tenantID,
			},
		})
		// tenant scoped platform + tenant scoped broker (broker and platform child)
		createOSBInstance(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, map[string]interface{}{
			"service_id":        "test-service",
			"plan_id":           "plan-service",
			"organization_guid": "my-org",
			"context": map[string]string{
				"tenant": tenantID,
			},
		})
		// osbInstanceID child
		createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding1", map[string]interface{}{
			"service_id":        "test-service",
			"plan_id":           "plan-service",
			"organization_guid": "my-org",
		})
		// osbInstanceID child
		createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding2", map[string]interface{}{
			"service_id":        "test-service",
			"plan_id":           "plan-service",
			"organization_guid": "my-org",
		})
	} else {
		ctx.SMWithBasic.SetBasicCredentials(ctx, tenantPlatformUser, tenantPlatformSecret)
	}
}

func createOSBInstance(ctx *TestContext, sm *SMExpect, brokerID string, instanceID string, osbContext map[string]interface{}) {
	subaccountResources[types.ServiceInstanceType]++
	smBrokerURL := ctx.Servers[SMServer].URL() + "/v1/osb/" + brokerID
	sm.PUT(smBrokerURL+"/v2/service_instances/"+instanceID).
		WithHeader("X-Broker-API-Version", "2.13").
		WithJSON(osbContext).
		WithQuery("accepts_incomplete", false).
		Expect().
		Status(http.StatusCreated)
}

func createSMAAPInstance(ctx *TestContext, sm *SMExpect, context map[string]interface{}) string {
	subaccountResources[types.ServiceInstanceType]++
	smBrokerURL := ctx.Servers[SMServer].URL()
	return sm.POST(smBrokerURL+web.ServiceInstancesURL).
		WithQuery("async", false).
		WithJSON(context).
		Expect().
		Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
}

func createSMAAPBinding(ctx *TestContext, sm *SMExpect, context map[string]interface{}) string {
	subaccountResources[types.ServiceBindingType]++
	smBrokerURL := ctx.Servers[SMServer].URL()
	return sm.POST(smBrokerURL+web.ServiceBindingsURL).
		WithQuery("async", false).
		WithJSON(context).
		Expect().
		Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
}

func createOSBBinding(ctx *TestContext, sm *SMExpect, brokerID string, instanceID string, bindingID string, osbContext map[string]interface{}) {
	subaccountResources[types.ServiceBindingType]++
	smBrokerURL := ctx.Servers[SMServer].URL() + "/v1/osb/" + brokerID
	sm.PUT(smBrokerURL+"/v2/service_instances/"+instanceID+"/service_bindings/"+bindingID).
		WithHeader("X-Broker-API-Version", "2.13").
		WithJSON(osbContext).
		WithQuery("accepts_incomplete", false).
		Expect().
		Status(http.StatusCreated)
}

func registertenantScopedBroker(ctx *TestContext, serviceNameID string, planID string) (string, *BrokerServer) {
	// registering a tenant scope broker
	catalog := SimpleCatalog(serviceNameID, planID, generateID())
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

	registerInstanceLastOPHandlers(brokerServer, http.StatusOK, types.SUCCEEDED)
	registerBindingLastOPHandlers(brokerServer, http.StatusOK, types.SUCCEEDED)
	CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, id)
	return id, brokerServer
}

func registerInstanceLastOPHandlers(brokerServer *BrokerServer, status int, state types.OperationState) {
	brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
		if status == http.StatusOK {
			return status, Object{"state": state}
		} else {
			return status, Object{}
		}
	})
}

func registerBindingLastOPHandlers(brokerServer *BrokerServer, status int, state types.OperationState) {
	brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", func(req *http.Request) (int, map[string]interface{}) {
		if status == http.StatusOK {
			return status, Object{"state": state}
		} else {
			return status, Object{}
		}
	})
}

func registertenantScopedPlatform(ctx *TestContext, name string) (string, string, string) {
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
	return id, username.Raw(), secret.Raw()
}

func SimpleCatalog(serviceID, planID string, planID2 string) SBCatalog {
	return SBCatalog(fmt.Sprintf(`{
    "services": [{
      "bindings_retrievable": true,
      "instances_retrievable": true,
      "name": "no-tags-no-metadata",
      "id": "%s",
      "description": "A fake service. ",
      "plans": [
      {
        "bindable": true,
        "name": "fake-plan-0",
        "id": "%s",
        "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections."
      },
      {
        "bindable": true,
        "name": "fake-plan-1",
        "id": "%s",
        "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections."
      }]
    }]
  }`, serviceID, planID2, planID))
}

func fetchFullTree(repository storage.TransactionalRepository, rootID string) (*tree, error) {
	fullTree := tree{
		allOperations:  make([]*types.Operation, 0),
		byResourceID:   make(map[string][]*types.Operation),
		byParentID:     make(map[string][]*types.Operation),
		byResourceType: make(map[types.ObjectType][]*types.Operation),
		byOperationID:  make(map[string]*types.Operation),
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
		fullTree.byResourceType[operation.ResourceType] = append(fullTree.byResourceType[operation.ResourceType], operation)
		fullTree.byOperationID[operation.ID] = operation
		fullTree.allOperations = append(fullTree.allOperations, operation)
	}

	testFullTree = &fullTree
	return &fullTree, nil
}

const (
	SucceededColor = "\033[48;5;193m%s\033[0m"
	FailedColor    = "\033[48;5;224m%s\033[0m"
)

func printFullTree(byParentID map[string][]*types.Operation, parent *types.Operation, spaces int) {
	for i := 0; i < spaces; i++ {
		fmt.Print("            ")
	}

	var newType string
	newType = strings.ReplaceAll(parent.ResourceType.String(), "/v1/", "")

	switch parent.State {
	case types.FAILED:
		fmt.Println(fmt.Sprintf(FailedColor, fmt.Sprintf(" - %s: %s ", newType, parent.ResourceID)))
	case types.SUCCEEDED:
		fmt.Println(fmt.Sprintf(SucceededColor, fmt.Sprintf(" - %s: %s ", newType, parent.ResourceID)))
	default:
		fmt.Println(fmt.Sprintf(" - %s: %s ", newType, parent.ResourceID))
	}
	for _, n := range byParentID[parent.ID] {
		printFullTree(byParentID, n, spaces+1)
	}
}

func AssertOperationCount(expect func(count int), criterion ...query.Criterion) {
	count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, criterion...)
	Expect(err).NotTo(HaveOccurred())
	expect(count)
}

type tree struct {
	allOperations  []*types.Operation
	root           *types.Operation
	byResourceID   map[string][]*types.Operation
	byResourceType map[types.ObjectType][]*types.Operation
	byParentID     map[string][]*types.Operation
	byOperationID  map[string]*types.Operation
}

func generateID() string {
	UUID, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	return UUID.String()
}

func triggerCascadeOperation(repoCtx context.Context, resourceType types.ObjectType, resourceID string, force bool) string {
	UUID, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	rootID := UUID.String()

	labels := map[string][]string{}
	if force {
		labels["force"] = []string{"true"}
	}

	cascadeOperation := types.Operation{
		Base: types.Base{
			ID:        rootID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Ready:     true,
			Labels:    labels,
		},
		Description:   "bla",
		CascadeRootID: rootID,
		ResourceID:    resourceID,
		Type:          types.DELETE,
		ResourceType:  resourceType,
	}
	_, err = ctx.SMRepository.Create(repoCtx, &cascadeOperation)
	//Last operation is never deleted since we keep the last operation for any resource (see: CleanupResourcelessOperations in maintainer.go).
	//That's why we create here another operation thus allow us to verify the deletion of the cascade operation in the tests.
	UUID, err = uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	lastOperation := types.Operation{
		Base:         types.Base{ID: UUID.String()},
		ResourceID:   resourceID,
		ResourceType: types.TenantType,
		Type:         types.CREATE,
		State:        "succeeded",
	}
	_, err = ctx.SMRepository.Create(repoCtx, &lastOperation)
	Expect(err).NotTo(HaveOccurred())
	return rootID
}

func validateAllResourceDeleted(repository storage.TransactionalRepository, byResourceType map[types.ObjectType][]*types.Operation) {
	By("validating resources have deleted")
	for objectType, operations := range byResourceType {
		if objectType != types.TenantType {
			IDs := make([]string, 0, len(operations))
			for _, operation := range operations {
				IDs = append(IDs, operation.ResourceID)
			}

			count, err := repository.Count(context.Background(), objectType, query.ByField(query.InOperator, "id", IDs...))
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(0), fmt.Sprintf("resources from type %s failed to be deleted", objectType))
		}
	}
}

type ContainerInstance struct {
	id                 string
	instances          []string
	bindingForInstance map[string][]string
}

func createContainerWithChildren() ContainerInstance {
	createOSBInstance(ctx, ctx.SMWithBasic, globalBrokerID, "container-instance", map[string]interface{}{
		"service_id":        "global-service",
		"plan_id":           "global-plan",
		"organization_guid": "my-org",
	})
	instanceInContainerID := createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
		"name":            "instance-in-container",
		"service_plan_id": plan.GetID(),
	})
	bindingInContainer := createSMAAPBinding(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
		"name":                "binding-in-container",
		"service_instance_id": instanceInContainerID,
	})

	containerInstance, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "name", "container-instance"))
	Expect(err).NotTo(HaveOccurred())
	instanceInContainer, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", instanceInContainerID))
	Expect(err).NotTo(HaveOccurred())

	change := types.LabelChange{
		Operation: "add",
		Key:       "containerID",
		Values:    []string{containerInstance.GetID()},
	}

	_, err = ctx.SMScheduler.ScheduleSyncStorageAction(context.TODO(), &types.Operation{
		Base: types.Base{
			ID:        "afasfasfasfasf",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Ready:     true,
		},
		Type:          types.UPDATE,
		State:         types.IN_PROGRESS,
		ResourceID:    instanceInContainer.GetID(),
		ResourceType:  types.ServiceInstanceType,
		CorrelationID: "-",
		Context:       &types.OperationContext{},
	}, func(ctx context.Context, repository storage.Repository) (object types.Object, e error) {
		return repository.Update(ctx, instanceInContainer, []*types.LabelChange{&change}, query.ByField(query.EqualsOperator, "id", instanceInContainer.GetID()))
	})
	Expect(err).NotTo(HaveOccurred())

	return ContainerInstance{
		id:                 "container-instance",
		instances:          []string{instanceInContainerID},
		bindingForInstance: map[string][]string{instanceInContainerID: {bindingInContainer}},
	}
}

func validateDuplicationHasTheSameState(fullTree *tree) {
	By("validating duplications waited and updated like sibling operations")
	for resourceID, operations := range fullTree.byResourceID {
		if resourceID == fullTree.root.ResourceID {
			continue
		}
		countOfOperationsThatProgressed := 0
		lastState := ""
		for _, operation := range operations {
			if operation.ExternalID != "" {
				countOfOperationsThatProgressed++
			}
			if lastState != "" {
				Expect(string(operation.State)).To(Equal(lastState))
			}
			lastState = string(operation.State)
		}
		Expect(countOfOperationsThatProgressed).
			To(Or(Equal(1), Equal(0)), fmt.Sprintf("resource: %s %s", operations[0].ResourceType, resourceID))
	}
}

func validateParentsRanAfterChildren(fullTree *tree) {
	By("validating parents ran after their children")
	for parentID, operations := range fullTree.byParentID {
		var parent *types.Operation
		if parentID == "" {
			parent = fullTree.root
		} else {
			parent = fullTree.byOperationID[parentID]
		}
		for _, operation := range operations {
			if !operation.DeletionScheduled.IsZero() {
				continue
			}
			Expect(parent.UpdatedAt.After(operation.UpdatedAt) || parent.UpdatedAt.Equal(operation.UpdatedAt)).
				To(BeTrue(), fmt.Sprintf("parent %s updateAt: %s is not after operation %s updateAt: %s", parent.ResourceType, parent.UpdatedAt, operation.ResourceType, operation.UpdatedAt))
		}
	}
}

func count(operations []*types.Operation, condition func(operation *types.Operation) bool) int {
	count := 0
	for _, operation := range operations {
		if condition(operation) {
			count++
		}
	}
	return count
}

func validateAllOperationsHasTheSameState(fullTree *tree, succeeded types.OperationState, operationsCount int) {
	By("validating all operations were succeeded")
	Expect(count(fullTree.allOperations, func(operation *types.Operation) bool {
		return operation.State == succeeded
	})).To(Equal(operationsCount))
}

func subaccountResourcesCount() int {
	return 2 * len(subaccountResources)
}
