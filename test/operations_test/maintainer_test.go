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
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOperations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Maintainer Tests Suite")
}

var _ = Describe("Maintainer", func() {

	const (
		jobTimeout      = 3 * time.Second
		cleanupInterval = 5 * time.Second
	)

	var ctx *common.TestContext

	BeforeSuite(func() {
		postHook := func(e env.Environment, servers map[string]common.FakeServer) {
			e.Set("operations.job_timeout", jobTimeout)
			e.Set("operations.cleanup_interval", cleanupInterval)
		}

		ctx = common.NewTestContextBuilder().WithEnvPostExtensions(postHook).Build()
	})

	AfterSuite(func() {
		ctx.Cleanup()
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
			operationID := "test-opration-id"
			operation := &types.Operation{
				Base: types.Base{
					ID:        operationID,
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
				byID := query.ByField(query.EqualsOperator, "id", operationID)
				object, err := ctx.SMRepository.Get(context.Background(), types.OperationType, byID)
				Expect(err).To(BeNil())

				op := object.(*types.Operation)
				return op.State
			}, jobTimeout*2).Should(Equal(types.FAILED))
		})
	})
})
