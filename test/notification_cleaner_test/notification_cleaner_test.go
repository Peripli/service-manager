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
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/test/testutil"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func TestNotificationCleaner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notification Cleaner Tests Suite")
}

var _ = Describe("Notification cleaner", func() {
	const (
		defaultKeepFor    = time.Hour * 6
		eventuallyTimeout = time.Second * 20
	)
	var (
		testContext    *common.TestContext
		ctx            context.Context
		logInterceptor *testutil.LogInterceptor
	)

	randomNotification := func() *types.Notification {
		idBytes, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		return &types.Notification{
			Base: types.Base{
				ID:        idBytes.String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Ready:     true,
			},
			Resource:   "resource",
			Type:       "CREATED",
			PlatformID: testContext.TestPlatform.ID,
		}
	}

	BeforeSuite(func() {
		logInterceptor = &testutil.LogInterceptor{}
		ctx = context.Background()

		testContext = common.NewTestContextBuilder().WithEnvPreExtensions(func(set *pflag.FlagSet) {
			err := set.Set("storage.notification.clean_interval", "10ms")
			Expect(err).ToNot(HaveOccurred())
			err = set.Set("storage.notification.keep_for", defaultKeepFor.String())
			Expect(err).ToNot(HaveOccurred())
		}).Build()
	})

	AfterSuite(func() {
		testContext.Cleanup()
	})

	Context("When two notifications are inserted", func() {
		It("Should delete the old one", func() {
			new, err := testContext.SMRepository.Create(ctx, randomNotification())
			Expect(err).ToNot(HaveOccurred())
			Expect(new.GetID()).ToNot(BeEmpty())

			oldNotification := randomNotification()
			oldNotification.CreatedAt = time.Now().Add(-(defaultKeepFor + time.Hour))

			log.AddHook(logInterceptor)
			old, err := testContext.SMRepository.Create(ctx, oldNotification)
			Expect(err).ToNot(HaveOccurred())
			Expect(old.GetID()).ToNot(BeNil())

			Eventually(logInterceptor.String, eventuallyTimeout).
				Should(ContainSubstring("successfully deleted notifications created before"))

			Eventually(func() error {
				byOldID := query.ByField(query.EqualsOperator, "id", old.GetID())
				_, err = testContext.SMRepository.Get(ctx, types.NotificationType, byOldID)
				return err
			}, eventuallyTimeout).Should(Equal(util.ErrNotFoundInStorage))

			byNewID := query.ByField(query.EqualsOperator, "id", new.GetID())
			obj, err := testContext.SMRepository.Get(ctx, types.NotificationType, byNewID)
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.GetID()).To(Equal(new.GetID()))
		})
	})
})
