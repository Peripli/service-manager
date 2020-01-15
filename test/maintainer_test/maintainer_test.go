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
package notification_cleaner_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/test/testutil"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func TestMaintainer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Maintainer Tests Suite")
}

var _ = Describe("Maintainer", func() {
	const (
		eventuallyTimeout = time.Second * 5
	)
	var (
		testContext    *common.TestContext
		ctx            context.Context
		logInterceptor *testutil.LogInterceptor
	)

	randomOperation := func() *types.Operation {
		idBytes, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		return &types.Operation{
			Base: types.Base{
				ID:        idBytes.String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			State: types.IN_PROGRESS,
			Type:  types.DELETE,
		}
	}

	AfterEach(func() {
		testContext.Cleanup()
	})

	BeforeEach(func() {
		logInterceptor = &testutil.LogInterceptor{}
		ctx = context.Background()

		testContext = common.NewTestContextBuilder().WithEnvPreExtensions(func(set *pflag.FlagSet) {
			err := set.Set("log.level", "debug")
			Expect(err).ToNot(HaveOccurred())
			err = set.Set("operations.default_pool_size", "1")
			Expect(err).ToNot(HaveOccurred())
			err = set.Set("operations.job_timeout", "10m")
			Expect(err).ToNot(HaveOccurred())
			err = set.Set("operations.mark_orphans_interval", "10ms")
			Expect(err).ToNot(HaveOccurred())
		}).Build()
	})

	Context("When three operations are created", func() {
		var first, second, third *types.Operation
		BeforeEach(func() {
			first = randomOperation()
			second = randomOperation()
			third = randomOperation()
		})

		JustBeforeEach(func() {
			var err error
			var new types.Object
			new, err = testContext.SMRepository.Create(ctx, first)
			Expect(err).ToNot(HaveOccurred())
			Expect(new.GetID()).ToNot(BeEmpty())
			first.SetID(new.GetID())

			new, err = testContext.SMRepository.Create(ctx, second)
			Expect(err).ToNot(HaveOccurred())
			Expect(new.GetID()).ToNot(BeEmpty())
			second.SetID(new.GetID())

			new, err = testContext.SMRepository.Create(ctx, third)
			Expect(err).ToNot(HaveOccurred())
			Expect(new.GetID()).ToNot(BeEmpty())
			third.SetID(new.GetID())
		})

		Context("when second is long in progress", func() {
			BeforeEach(func() {
				second.CreatedAt = time.Now().Add(-(time.Hour))
			})

			It("should mark the second as failed", func() {
				log.AddHook(logInterceptor)

				Eventually(logInterceptor.String, eventuallyTimeout).
					Should(ContainSubstring(fmt.Sprintf("Successfully marked orphan operation with id %s", second.ID)))

				Eventually(func() types.OperationState {
					byOldID := query.ByField(query.EqualsOperator, "id", second.GetID())
					object, err := testContext.SMRepository.Get(ctx, types.OperationType, byOldID)
					Expect(err).ShouldNot(HaveOccurred())
					return (object.(*types.Operation)).State
				}, eventuallyTimeout).Should(Equal(types.FAILED))

				Eventually(func() types.OperationState {
					byOldID := query.ByField(query.EqualsOperator, "id", first.GetID())
					object, err := testContext.SMRepository.Get(ctx, types.OperationType, byOldID)
					Expect(err).ShouldNot(HaveOccurred())
					return (object.(*types.Operation)).State
				}, eventuallyTimeout).Should(Equal(types.IN_PROGRESS))
			})
		})

		Context("when maintainer is busy with other operations", func() {
			BeforeEach(func() {
				first.CreatedAt = time.Now().Add(-(time.Hour))
				second.CreatedAt = time.Now().Add(-2 * (time.Hour))
				third.CreatedAt = time.Now().Add(-3 * (time.Hour))
			})

			It("should log an appropriate error", func() {
				log.AddHook(logInterceptor)

				Eventually(logInterceptor.String, eventuallyTimeout).
					Should(ContainSubstring(fmt.Sprintf("Successfully marked orphan operation with id %s", second.ID)))

				Eventually(logInterceptor.String, eventuallyTimeout).
					Should(ContainSubstring(fmt.Sprintf("Maintainer too busy. Will schedule operation with id %s next time", third.ID)))

				Eventually(func() types.OperationState {
					byOldID := query.ByField(query.EqualsOperator, "id", second.GetID())
					object, err := testContext.SMRepository.Get(ctx, types.OperationType, byOldID)
					Expect(err).ShouldNot(HaveOccurred())
					return (object.(*types.Operation)).State
				}, eventuallyTimeout).Should(Equal(types.FAILED))
			})
		})

	})
})
