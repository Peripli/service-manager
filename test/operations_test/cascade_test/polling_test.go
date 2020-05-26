package cascade_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/multitenancy"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	. "github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestFilters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cascade Test Suite")
}

var _ = Describe("Cascade", func() {
	var (
		ctx          *TestContext
		brokerServer *BrokerServer
		brokerID     string
		plan         types.Object
		platformID   string
	)

	const (
		polling         = 1 * time.Millisecond
		maintainerRetry = 1 * time.Second
		actionTimeout   = 2 * time.Second
		cleanupInterval = 9999 * time.Hour
		pollCascade     = 500 * time.Millisecond
		reconciliation  = 9999 * time.Hour
		lifespan        = 1 * time.Millisecond
	)

	postHookWithOperationsConfig := func() func(e env.Environment, servers map[string]FakeServer) {
		return func(e env.Environment, servers map[string]FakeServer) {
			e.Set("operations.action_timeout", actionTimeout)
			e.Set("operations.maintainer_retry_interval", maintainerRetry)
			e.Set("operations.polling_interval", polling)
			e.Set("operations.cleanup_interval", cleanupInterval)
			e.Set("operations.poll_cascade_interval", pollCascade)
			e.Set("operations.lifespan", lifespan)
			e.Set("operations.reconciliation_operation_timeout", reconciliation)
		}
	}

	assertOperationCount := func(expect func(count int), criterion ...query.Criterion) {
		count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, criterion...)
		Expect(err).NotTo(HaveOccurred())
		expect(count)
	}

	queryForOperationsInTheSameTree := query.ByField(query.EqualsOperator, "cascade_root_id", "op1")
	queryForRoot := query.ByField(query.EqualsOperator, "id", "op1")
	//queryForInstanceOperations := query.ByField(query.EqualsOperator, "resource_type", types.ServiceInstanceType.String())
	//queryForBindingsOperations := query.ByField(query.EqualsOperator, "resource_type", types.ServiceBindingType.String())
	//queryForPlatformOperations := query.ByField(query.EqualsOperator, "resource_type", types.PlatformType.String())
	//queryForTenantOperations := query.ByField(query.EqualsOperator, "resource_type", types.TenantType.String())
	//queryForBrokerOperations := query.ByField(query.EqualsOperator, "resource_type", types.ServiceBrokerType.String())

	querySucceeded := query.ByField(query.EqualsOperator, "state", string(types.SUCCEEDED))
	//queryInProgress := query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS))
	//queryPending := query.ByField(query.EqualsOperator, "state", string(types.PENDING))
	queryFailedOperations := query.ByField(query.EqualsOperator, "state", string(types.FAILED))

	//queryForDuplications := query.ByField(query.EqualsOrNilOperator, "external_id", "")

	BeforeEach(func() {
		postHook := postHookWithOperationsConfig()
		ctx = NewTestContextBuilderWithSecurity().WithEnvPostExtensions(postHook).
			WithEnvPreExtensions(func(set *pflag.FlagSet) {
				Expect(set.Set("server.request_timeout", "4s")).ToNot(HaveOccurred())
				Expect(set.Set("httpclient.response_header_timeout", (time.Millisecond * 1500).String())).ToNot(HaveOccurred())
				Expect(set.Set("httpclient.timeout", (time.Millisecond * 1500).String())).ToNot(HaveOccurred())
			}).
			WithTenantTokenClaims(map[string]interface{}{
				"cid": "tenancyClient",
				"zid": "tenant_value",
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

	Context("Cascade Delete", func() {
		var beforeTestOperationsCount = 1 // virtual op for tenant
		BeforeEach(func() {
			brokerID, brokerServer = registerSubaccountScopedBroker(ctx, "test-service", "plan-service")
			beforeTestOperationsCount += 1
			platformID = registerSubaccountScopedPlatform(ctx, "platform1")
			beforeTestOperationsCount += 1
			var err error
			plan, err = ctx.SMRepository.Get(context.Background(), types.ServicePlanType, query.ByField(query.EqualsOperator, "catalog_id", "plan-service"))
			Expect(err).NotTo(HaveOccurred())
			createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
				"name":            "test-instance-smaap",
				"service_plan_id": plan.GetID(),
			})
			beforeTestOperationsCount += 2 //one for platform and one for broker
			createOSBInstance(ctx, ctx.SMWithBasic, brokerID, "test-instance", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})
			beforeTestOperationsCount += 2 //one for platform and one for broker
			createOSBBinding(ctx, ctx.SMWithBasic, brokerID, "test-instance", "binding1", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})
			beforeTestOperationsCount += 2 //one for platform and one for broker
			createOSBBinding(ctx, ctx.SMWithBasic, brokerID, "test-instance", "binding2", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})
			beforeTestOperationsCount += 2 //one for platform and one for broker
		})

		AfterEach(func() {
			ctx.Cleanup()
		})

		It("big tenant tree should succeed", func() {
			subtreeCount := 5
			for i := 0; i < subtreeCount; i++ {
				registerSubaccountScopedPlatform(ctx, fmt.Sprintf("platform%s", strconv.Itoa(i*10)))
				broker2ID, _ := registerSubaccountScopedBroker(ctx, fmt.Sprintf("test-service%s", strconv.Itoa(i*10)), fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)))
				createSMAAPInstance(ctx, ctx.SMWithOAuthForTenant, map[string]interface{}{
					"name":            fmt.Sprintf("test-instance-smaap%s", strconv.Itoa(i*10)),
					"service_plan_id": plan.GetID(),
				})
				createOSBInstance(ctx, ctx.SMWithBasic, broker2ID, fmt.Sprintf("test-instance%s", strconv.Itoa(i*10)), map[string]interface{}{
					"service_id":        fmt.Sprintf("test-service%s", strconv.Itoa(i*10)),
					"plan_id":           fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)),
					"organization_guid": "my-org",
				})
				createOSBBinding(ctx, ctx.SMWithBasic, broker2ID, fmt.Sprintf("test-instance%s", strconv.Itoa(i*10)), fmt.Sprintf("binding%s", strconv.Itoa((i+1)*10)), map[string]interface{}{
					"service_id":        fmt.Sprintf("test-service%s", strconv.Itoa(i*10)),
					"plan_id":           fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)),
					"organization_guid": "my-org",
				})
				createOSBBinding(ctx, ctx.SMWithBasic, broker2ID, fmt.Sprintf("test-instance%s", strconv.Itoa(i*10)), fmt.Sprintf("binding%s", strconv.Itoa((i+1)*10+1)), map[string]interface{}{
					"service_id":        fmt.Sprintf("test-service%s", strconv.Itoa(i*10)),
					"plan_id":           fmt.Sprintf("plan-service%s", strconv.Itoa(i*10)),
					"organization_guid": "my-org",
				})
			}

			op := types.Operation{
				Base: types.Base{
					ID:        "op1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Description:   "bla",
				CascadeRootID: "op1",
				ResourceID:    "tenant_value",
				State:         types.IN_PROGRESS,
				Type:          types.DELETE,
				ResourceType:  types.TenantType,
			}
			_, err := ctx.SMRepository.Create(context.TODO(), &op)
			Expect(err).NotTo(HaveOccurred())

			assertOperationCount(func(count int) { Expect(count).To(Equal(3 + subtreeCount*3)) }, query.ByField(query.EqualsOperator, "parent_id", "op1"))
			assertOperationCount(func(count int) { Expect(count).To(Equal(beforeTestOperationsCount + subtreeCount*10)) }, queryForOperationsInTheSameTree)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchAFullTree(ctx.SMRepository, "op1")
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})

		It("platform tree should succeed", func() {
			op := types.Operation{
				Base: types.Base{
					ID:        "op1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Description:   "bla",
				CascadeRootID: "op1",
				ResourceID:    platformID,
				State:         types.IN_PROGRESS,
				Type:          types.DELETE,
				ResourceType:  types.PlatformType,
			}
			_, err := ctx.SMRepository.Create(context.TODO(), &op)
			Expect(err).NotTo(HaveOccurred())

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchAFullTree(ctx.SMRepository, "op1")
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)
		})

		It("platform with container instance should fail", func() {
			registerInstanceLastOPHandlers(brokerServer, "failed")
			createOSBInstance(ctx, ctx.SMWithBasic, brokerID, "container-instance", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})
			createOSBInstance(ctx, ctx.SMWithBasic, brokerID, "child-instance", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})

			containerInstance, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "name", "container-instance"))
			Expect(err).NotTo(HaveOccurred())
			instanceInContainer, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "name", "child-instance"))
			Expect(err).NotTo(HaveOccurred())

			change := types.LabelChange{
				Operation: "add",
				Key:       "containerID",
				Values:    []string{containerInstance.GetID()},
			}
			_, err = ctx.SMRepository.Update(context.Background(), instanceInContainer, []*types.LabelChange{&change}, query.ByField(query.EqualsOperator, "id", instanceInContainer.GetID()))
			Expect(err).NotTo(HaveOccurred())

			op := types.Operation{
				Base: types.Base{
					ID:        "op1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Ready:     true,
				},
				Description:   "bla",
				CascadeRootID: "op1",
				ResourceID:    platformID,
				State:         types.IN_PROGRESS,
				Type:          types.DELETE,
				ResourceType:  types.PlatformType,
			}
			newCtx := context.WithValue(context.Background(), cascade.ContainerKey{}, "containerID")
			_, err = ctx.SMRepository.Create(newCtx, &op)
			Expect(err).NotTo(HaveOccurred())

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					queryFailedOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchAFullTree(ctx.SMRepository, "op1")
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)

			By("validating containerized errors collected")
			platformOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot)
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(platformOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(2))

			count := 0
			for _, e := range errors.Errors {
				if e.ParentType == types.ServiceInstanceType && e.ResourceType == types.ServiceInstanceType {
					count++
				}
			}
			Expect(count).To(Equal(1))
		})

		It("validate errors aggregated from bottom up", func() {
			registerBindingLastOPHandlers(brokerServer, types.FAILED)

			op := types.Operation{
				Base: types.Base{
					ID:        "op1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Description:   "bla",
				CascadeRootID: "op1",
				ResourceID:    "tenant_value",
				State:         types.IN_PROGRESS,
				Type:          types.DELETE,
				ResourceType:  types.TenantType,
			}
			_, err := ctx.SMRepository.Create(context.TODO(), &op)
			Expect(err).NotTo(HaveOccurred())

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot,
					queryFailedOperations)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*11+pollCascade*11).Should(Equal(1))

			fullTree, err := fetchAFullTree(ctx.SMRepository, "op1")
			Expect(err).NotTo(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsWaited(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot)
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(2))

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Equal("test-instance"))
				Expect(e.ResourceID).To(Or(Equal("binding1"), Equal("binding2")))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Equal(types.ServiceBindingType))
				Expect(e.ParentType).To(Equal(types.ServiceInstanceType))
			}
		})

		It("Cascade container", func() {

		})

		It("Activate a orphan mitigation for instance and expect for failures", func() {

		})

		It("Handle a stuck operation in cascade tree", func() {

		})
	})
})

func validateDuplicationsWaited(fullTree *tree) {
	By("validating duplications waited and updated like sibling operators")
	for resourceID, operations := range fullTree.byResourceID {
		if resourceID == fullTree.root.ResourceID {
			continue
		}
		countOfOperationsThatProgressed := 0
		for _, operation := range operations {
			if operation.ExternalID != "" {
				countOfOperationsThatProgressed++
			}
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
	raw := sm.POST(smBrokerURL+web.ServiceBindingsURL).
		WithQuery("async", false).
		WithJSON(context).
		Expect().
		JSON().Raw()

	if raw == nil {
	}
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
				"name": "fake-plan-1",
				"id": "%s",
				"description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections."
			}]
		}]
	}`, serviceID, planID))
}

func fetchAFullTree(repository storage.TransactionalRepository, rootID string) (*tree, error) {
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

type tree struct {
	root          *types.Operation
	byResourceID  map[string][]*types.Operation
	byParentID    map[string][]*types.Operation
	byOperationID map[string]*types.Operation
}
