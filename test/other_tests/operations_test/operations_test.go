/*
 *    Copyright 2018 The Service Manager Authors
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
package operations_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/test/common"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Peripli/service-manager/test"
	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	defaultOperationID = "test-operation-id"
	testControllerURL  = "/v1/panic"
)

func TestOperations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operations Tests Suite")
}

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.OperationsURL,
	SupportedOps: []test.Op{
		test.List, test.Delete,
	},
	DisableTenantResources:                 true,
	DisableBasicAuth:                       true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	PatchResource:                          test.StorageResourcePatch,
	ResourcePropertiesToIgnore:             []string{"transitive_resources"},
	AdditionalTests: func(ctx *common.TestContext, t *test.TestCase) {
		Describe("Operations", func() {
			var ctx *common.TestContext

			AfterEach(func() {
				ctx.Cleanup()
			})

			Context("Scheduler", func() {
				postBrokerBody := func() common.Object {
					brokerServer := common.NewBrokerServer()
					ctx.Servers[common.BrokerServerPrefix+"123"] = brokerServer
					return common.Object{
						"id":         "123",
						"name":       "test-broker",
						"broker_url": brokerServer.URL(),
						"credentials": common.Object{
							"basic": common.Object{
								"username": brokerServer.Username,
								"password": brokerServer.Password,
							},
						},
					}
				}

				for isAsync := range []string{"true", "false"} {
					When("job timeout runs out", func() {
						BeforeEach(func() {
							postHook := func(e env.Environment, servers map[string]common.FakeServer) {
								e.Set("operations.action_timeout", 6*time.Nanosecond)
							}

							ctx = common.NewTestContextBuilder().WithEnvPostExtensions(postHook).Build()
						})

						It("marks operation as failed", func() {
							resp := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerBody()).
								WithQuery("async", "true").
								Expect()

							common.VerifyOperationExists(ctx, resp.Header("Location").Raw(), common.OperationExpectations{
								Category:          types.CREATE,
								State:             types.FAILED,
								ResourceType:      types.ServiceBrokerType,
								Reschedulable:     false,
								DeletionScheduled: false,
								Error:             "could not reach service broker",
							})
						})
					})

					When("reconciliation timeout runs out", func() {
						BeforeEach(func() {
							postHook := func(e env.Environment, servers map[string]common.FakeServer) {
								e.Set("operations.reconciliation_operation_timeout", 5*time.Nanosecond)
							}

							ctx = common.NewTestContextBuilder().WithEnvPostExtensions(postHook).SkipBasicAuthClientSetup(true).Build()
						})

						It("marks operation as failed", func() {
							ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerBody()).
								WithQuery("async", isAsync).
								Expect().
								Status(http.StatusUnprocessableEntity)
						})
					})
				}

				When("when there are no available workers", func() {
					BeforeEach(func() {
						ctx = common.NewTestContextBuilder().Build()
					})

					It("returns 503", func() {
						brokerServer := common.NewBrokerServer()
						postBrokerRequestWithNoLabels := common.Object{
							"name":       "test-broker",
							"broker_url": brokerServer.URL(),
							"credentials": common.Object{
								"basic": common.Object{
									"username": brokerServer.Username,
									"password": brokerServer.Password,
								},
							},
						}

						requestCount := 100
						resultChan := make(chan struct{}, requestCount)
						executeReq := func() {
							resp := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
								WithQuery("async", "true").Expect()
							if resp.Raw().StatusCode == http.StatusServiceUnavailable {
								resultChan <- struct{}{}
							}
						}

						for i := 0; i < requestCount; i++ {
							go executeReq()
						}

						ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 6*time.Second)
						defer cancel()

						select {
						case <-ctxWithTimeout.Done():
							Fail("expected to get 503 from server due to too many async requests but didn't")
						case <-resultChan:
						}
					})
				})
			})

			Context("Jobs", func() {

				var operation *types.Operation

				BeforeEach(func() {
					operation = &types.Operation{
						Base: types.Base{
							ID:        defaultOperationID,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							Labels:    make(map[string][]string),
							Ready:     true,
						},
						Description:       "",
						Type:              types.CREATE,
						State:             types.IN_PROGRESS,
						ResourceID:        "test-resource-id",
						ResourceType:      web.ServiceBrokersURL,
						CorrelationID:     "test-correlation-id",
						Reschedule:        false,
						DeletionScheduled: time.Time{},
					}

					ctx = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
						testController := panicController{
							operation: operation,
							scheduler: operations.NewScheduler(ctx, smb.Storage, operations.DefaultSettings(), 10, &sync.WaitGroup{}),
						}

						smb.RegisterControllers(testController)
						return nil
					}).Build()

				})

				When("job panics", func() {
					It("recovers successfully and marks operation as failed", func() {
						ctx.SM.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)
						ctx.SM.GET(testControllerURL).Expect()
						ctx.SM.GET(web.MonitorHealthURL).Expect().Status(http.StatusOK)

						var respBody *httpexpect.Object
						Eventually(func() string {
							respBody = ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s%s/%s", operation.ResourceType, operation.ResourceID, web.ResourceOperationsURL, operation.ID)).
								Expect().Status(http.StatusOK).JSON().Object()
							return respBody.Value("state").String().Raw()
						}, 2*time.Second).Should(Equal("failed"))

						Expect(respBody.Value("errors").Object().Value("description").String().Raw()).To(ContainSubstring("job interrupted"))
					})
				})
			})

			Context("Maintainer", func() {
				const (
					maintainerRetry     = 1 * time.Second
					actionTimeout       = 2 * time.Second
					cleanupInterval     = 3 * time.Second
					operationExpiration = 3 * time.Second
				)

				var ctxBuilder *common.TestContextBuilder

				postHookWithOperationsConfig := func() func(e env.Environment, servers map[string]common.FakeServer) {
					return func(e env.Environment, servers map[string]common.FakeServer) {
						e.Set("operations.action_timeout", actionTimeout)
						e.Set("operations.maintainer_retry_interval", maintainerRetry)
						e.Set("operations.cleanup_interval", cleanupInterval)
						e.Set("operations.lifespan", operationExpiration)
						e.Set("operations.reconciliation_operation_timeout", 9999*time.Hour)
					}
				}

				assertOperationCount := func(expectedCount int, criterion ...query.Criterion) {
					count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, criterion...)
					Expect(err).To(BeNil())
					Expect(count).To(Equal(expectedCount))
				}

				BeforeEach(func() {
					postHook := postHookWithOperationsConfig()
					ctxBuilder = common.NewTestContextBuilderWithSecurity().WithEnvPostExtensions(postHook)
				})

				When("Specified cleanup interval passes", func() {
					BeforeEach(func() {
						ctx = ctxBuilder.Build()
					})

					Context("operation platform is service Manager", func() {
						It("Deletes operations older than that interval", func() {
							ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL+"/non-existent-broker-id").WithQuery("async", true).
								Expect().
								Status(http.StatusAccepted)

							byPlatformID := query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform)
							assertOperationCount(2, byPlatformID)

							Eventually(func() int {
								count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, byPlatformID)
								Expect(err).To(BeNil())

								return count
							}, cleanupInterval*4).Should(Equal(0))
						})
					})

					Context("operation platform is platform registered in service manager", func() {
						const (
							brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
							brokerAPIVersionHeaderValue = "2.13"

							serviceID = "test-service-1"
							planID    = "test-service-plan-1"
						)

						var (
							brokerID     string
							catalog      common.SBCatalog
							brokerServer *common.BrokerServer
						)

						asyncProvision := func() {
							ctx.SMWithBasic.PUT(brokerServer.URL()+"/v1/osb/"+brokerID+"/v2/service_instances/12345").
								WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
								WithQuery("async", true).
								WithJSON(map[string]interface{}{
									"service_id":        serviceID,
									"plan_id":           planID,
									"organization_guid": "my-org",
								}).
								Expect().Status(http.StatusAccepted)
						}

						BeforeEach(func() {
							catalog = simpleCatalog(serviceID, planID)
							catalog = simpleCatalog(serviceID, planID)
							ctx.RegisterPlatform()
							brokerID, _, brokerServer = ctx.RegisterBrokerWithCatalog(catalog).GetBrokerAsParams()
							brokerServer.ServiceInstanceHandler = func(rw http.ResponseWriter, _ *http.Request) {
								rw.Header().Set("Content-Type", "application/json")
								rw.WriteHeader(http.StatusAccepted)
							}
							common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, brokerID)
						})

						AfterEach(func() {
							common.RemoveAllInstances(ctx)
							ctx.CleanupBroker(brokerID)
						})

						It("Deletes operations older than that interval", func() {
							asyncProvision()
							byPlatformID := query.ByField(query.NotEqualsOperator, "platform_id", types.SMPlatform)

							assertOperationCount(1, byPlatformID)

							Eventually(func() int {
								count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, byPlatformID)
								Expect(err).To(BeNil())

								return count
							}, cleanupInterval*4).Should(Equal(0))
						})
					})

					Context("with external operations for Service Manager", func() {
						BeforeEach(func() {
							operation := &types.Operation{
								Base: types.Base{
									ID:        defaultOperationID,
									UpdatedAt: time.Now().Add(-cleanupInterval + time.Second),
									Labels:    make(map[string][]string),
									Ready:     true,
								},
								Reschedule:    false,
								Type:          types.CREATE,
								State:         types.IN_PROGRESS,
								ResourceID:    "test-resource-id",
								ResourceType:  web.ServiceBrokersURL,
								PlatformID:    "cloudfoundry",
								CorrelationID: "test-correlation-id",
							}
							object, err := ctx.SMRepository.Create(context.Background(), operation)
							Expect(err).To(BeNil())
							Expect(object).To(Not(BeNil()))
						})

						It("should cleanup external old ones", func() {
							byPlatformID := query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform)
							assertOperationCount(1, byPlatformID)
							Eventually(func() int {
								count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, byPlatformID)
								Expect(err).To(BeNil())

								return count
							}, operationExpiration*3).Should(Equal(0))
						})
					})
				})

				When("Specified action timeout passes", func() {
					BeforeEach(func() {
						ctx = ctxBuilder.Build()
					})

					It("Marks orphans as failed operations", func() {
						operation := &types.Operation{
							Base: types.Base{
								ID:        defaultOperationID,
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
								Labels:    make(map[string][]string),
								Ready:     true,
							},
							Reschedule:    false,
							Type:          types.CREATE,
							State:         types.IN_PROGRESS,
							ResourceID:    "test-resource-id",
							ResourceType:  web.ServiceBrokersURL,
							PlatformID:    types.SMPlatform,
							CorrelationID: "test-correlation-id",
						}

						object, err := ctx.SMRepository.Create(context.Background(), operation)
						Expect(err).To(BeNil())
						Expect(object).To(Not(BeNil()))

						Eventually(func() types.OperationState {
							byID := query.ByField(query.EqualsOperator, "id", defaultOperationID)
							object, err := ctx.SMRepository.Get(context.Background(), types.OperationType, byID)
							Expect(err).To(BeNil())

							op := object.(*types.Operation)
							return op.State
						}, actionTimeout*6).Should(Equal(types.FAILED))
					})
				})

				When("operation gets stuck in progress without being reschedulable", func() {
					var operation *types.Operation

					BeforeEach(func() {
						ctxBuilder.WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
							smb.WithDeleteAroundTxInterceptorProvider(types.ServiceBrokerType, DeleteBrokerDelayingInterceptorProvider{}).Register()
							return nil
						})

						ctx = ctxBuilder.Build()

						operation = &types.Operation{
							Base: types.Base{
								ID:        defaultOperationID,
								CreatedAt: time.Now().Add(-5 * actionTimeout),
								UpdatedAt: time.Now().Add(-5 * actionTimeout),
								Labels:    make(map[string][]string),
								Ready:     true,
							},
							State:         types.IN_PROGRESS,
							ResourceID:    "test-resource-id",
							ResourceType:  web.ServiceBrokersURL,
							PlatformID:    types.SMPlatform,
							CorrelationID: "test-correlation-id",
							Reschedule:    false,
						}
					})

					When("when operation is create", func() {
						It("marks the operation as failed with scheduled deletion", func() {
							operation.Type = types.CREATE

							object, err := ctx.SMRepository.Create(context.Background(), operation)
							Expect(err).To(BeNil())
							Expect(object).To(Not(BeNil()))

							common.VerifyOperationExists(ctx, "", common.OperationExpectations{
								Category:          operation.Type,
								State:             types.FAILED,
								ResourceType:      operation.ResourceType,
								Reschedulable:     false,
								DeletionScheduled: true,
							})
						})
					})

					When("when operation is delete", func() {
						It("marks the operation as failed with scheduled deletion", func() {
							operation.Type = types.DELETE

							object, err := ctx.SMRepository.Create(context.Background(), operation)
							Expect(err).To(BeNil())
							Expect(object).To(Not(BeNil()))

							common.VerifyOperationExists(ctx, "", common.OperationExpectations{
								Category:          operation.Type,
								State:             types.FAILED,
								ResourceType:      operation.ResourceType,
								Reschedulable:     false,
								DeletionScheduled: true,
							})
						})
					})

					When("when operation is update", func() {
						It("marks the operation as failed without scheduled deletion", func() {
							operation.Type = types.UPDATE

							object, err := ctx.SMRepository.Create(context.Background(), operation)
							Expect(err).To(BeNil())
							Expect(object).To(Not(BeNil()))

							common.VerifyOperationExists(ctx, "", common.OperationExpectations{
								Category:          operation.Type,
								State:             types.FAILED,
								ResourceType:      operation.ResourceType,
								Reschedulable:     false,
								DeletionScheduled: false,
							})
						})
					})
				})
			})
		})
	},
})

type DeleteBrokerDelayingInterceptorProvider struct{}

func (DeleteBrokerDelayingInterceptorProvider) Name() string {
	return "DeleteBrokerDelayingInterceptorProvider"
}

func (DeleteBrokerDelayingInterceptorProvider) Provide() storage.DeleteAroundTxInterceptor {
	return DeleteBrokerDelayingInterceptor{}
}

type DeleteBrokerDelayingInterceptor struct{}

func (DeleteBrokerDelayingInterceptor) AroundTxDelete(f storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	<-time.After(2 * time.Second)
	return f
}

func blueprint(ctx *common.TestContext, _ *common.SMExpect, _ bool) common.Object {
	cPaidPlan := common.GeneratePaidTestPlan()
	cService := common.GenerateTestServiceWithPlans(cPaidPlan)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)

	brokerServer := common.NewBrokerServerWithCatalog(catalog)
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	UUID2, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	brokerJSON := common.Object{
		"name":        UUID.String(),
		"broker_url":  brokerServer.URL(),
		"description": UUID2.String(),
		"credentials": common.Object{
			"basic": common.Object{
				"username": brokerServer.Username,
				"password": brokerServer.Password,
			},
		},
	}
	resp := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(brokerJSON).
		WithQuery("async", "true").
		Expect().
		Status(http.StatusAccepted)

	ctx.Servers[common.BrokerServerPrefix+UUID.String()] = brokerServer

	operationURL := resp.Header("Location").Raw()

	common.VerifyOperationExists(ctx, operationURL, common.OperationExpectations{
		Category:          types.CREATE,
		State:             types.SUCCEEDED,
		ResourceType:      types.ServiceBrokerType,
		Reschedulable:     false,
		DeletionScheduled: false,
	})

	return ctx.SMWithOAuth.GET(operationURL).Expect().Status(http.StatusOK).JSON().Object().Raw()
}

func simpleCatalog(serviceID, planID string) common.SBCatalog {
	return common.SBCatalog(fmt.Sprintf(`{
	  "services": [{
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

type panicController struct {
	operation *types.Operation
	scheduler *operations.Scheduler
}

func (pc panicController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   testControllerURL,
			},
			Handler: func(req *web.Request) (resp *web.Response, err error) {
				pc.scheduler.ScheduleAsyncStorageAction(context.TODO(), pc.operation, func(ctx context.Context, repository storage.Repository) (object types.Object, e error) {
					panic("test panic")
				})
				return
			},
		},
	}

}
