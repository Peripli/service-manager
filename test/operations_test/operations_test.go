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
	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	defaultOperationID = "test-operation-id"
	testControllerURL  = "/v1/panic"
)

func TestOperations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operations Tests Suite")
}

var _ = Describe("Operations", func() {

	var ctx *common.TestContext

	AfterEach(func() {
		ctx.Cleanup()
	})

	Context("Scheduler", func() {
		BeforeEach(func() {
			postHook := func(e env.Environment, servers map[string]common.FakeServer) {
				e.Set("operations.job_timeout", 2*time.Nanosecond)
				e.Set("operations.mark_orphans_interval", 1*time.Hour)
			}

			ctx = common.NewTestContextBuilder().WithEnvPostExtensions(postHook).Build()
		})

		When("job timeout runs out", func() {
			It("marks operation as failed", func() {
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

				resp := ctx.SMWithOAuth.POST(web.ServiceBrokersURL).WithJSON(postBrokerRequestWithNoLabels).
					WithQuery("async", "true").
					Expect().
					Status(http.StatusAccepted)
				_, err := test.ExpectOperationWithError(ctx.SMWithOAuth, resp, types.FAILED, "job timed out")
				Expect(err).To(BeNil())
			})
		})

		When("when there are no available workers", func() {
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

				ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
				},
				Type:          types.CREATE,
				State:         types.IN_PROGRESS,
				ResourceID:    "test-resource-id",
				ResourceType:  web.ServiceBrokersURL,
				CorrelationID: "test-correlation-id",
			}

			ctx = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				testController := panicController{
					operation: operation,
					scheduler: operations.NewScheduler(ctx, smb.Storage, 10*time.Minute, 10, &sync.WaitGroup{}),
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
					respBody = ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s%s/%s", operation.ResourceType, operation.ResourceID, web.OperationsURL, operation.ID)).
						Expect().Status(http.StatusOK).JSON().Object()
					return respBody.Value("state").String().Raw()
				}, 2*time.Second).Should(Equal("failed"))

				Expect(respBody.Value("errors").Object().Value("message").String().Raw()).To(ContainSubstring("job interrupted"))
			})
		})
	})

	Context("Maintainer", func() {
		const (
			jobTimeout      = 3 * time.Second
			cleanupInterval = 5 * time.Second
		)

		BeforeEach(func() {
			postHook := func(e env.Environment, servers map[string]common.FakeServer) {
				e.Set("operations.job_timeout", jobTimeout)
				e.Set("operations.mark_orphans_interval", jobTimeout)
				e.Set("operations.cleanup_interval", cleanupInterval)
			}

			ctx = common.NewTestContextBuilder().WithEnvPostExtensions(postHook).Build()
		})

		When("Specified cleanup interval passes", func() {
			It("Deletes operations older than that interval", func() {
				resp := ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL+"/non-existent-broker-id").WithQuery("async", true).
					Expect().
					Status(http.StatusAccepted)

				locationHeader := resp.Header("Location").Raw()
				split := strings.Split(locationHeader, "/")
				operationID := split[len(split)-1]

				byID := query.ByField(query.EqualsOperator, "id", operationID)
				count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, byID)
				Expect(err).To(BeNil())
				Expect(count).To(Equal(1))

				Eventually(func() int {
					count, err := ctx.SMRepository.Count(context.Background(), types.OperationType, byID)
					Expect(err).To(BeNil())

					return count
				}, cleanupInterval*2).Should(Equal(0))
			})

		})

		When("Specified job timeout passes", func() {
			It("Marks orphans as failed operations", func() {
				operation := &types.Operation{
					Base: types.Base{
						ID:        defaultOperationID,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Labels:    make(map[string][]string),
					},
					Type:          types.CREATE,
					State:         types.IN_PROGRESS,
					ResourceID:    "test-resource-id",
					ResourceType:  web.ServiceBrokersURL,
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
				}, jobTimeout*5).Should(Equal(types.FAILED))
			})
		})
	})
})

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
				job := operations.Job{
					ReqCtx:     context.Background(),
					ObjectType: "test-type",
					Operation:  pc.operation,
					OperationFunc: func(ctx context.Context, repository storage.Repository) (object types.Object, e error) {
						panic("test panic")
					},
				}

				pc.scheduler.Schedule(job)
				return
			},
		},
	}

}
