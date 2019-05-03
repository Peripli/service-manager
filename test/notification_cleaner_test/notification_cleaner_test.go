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

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func TestNotificationCleaner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notification Cleaner Tests Suite")
}

var _ = Describe("Notification cleaner", func() {
	var testContext *common.TestContext
	var repository storage.Repository
	var defaultOlderThan time.Duration
	var ctx context.Context

	randomNotification := func() *types.Notification {
		idBytes, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		return &types.Notification{
			Base: types.Base{
				ID:        idBytes.String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Resource: "resource",
			Type:     "CREATED",
		}
	}

	BeforeSuite(func() {
		ctx = context.Background()
		defaultOlderThan = storage.DefaultSettings().Notification.OlderThan
		testContext = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			repository = smb.Storage
			// TODO add channel which signals storage delete
			smb.WithDeleteInterceptorProvider(types.NotificationType, nil).Register()
			return nil
		}).WithEnvPreExtensions(func(set *pflag.FlagSet) {
			err := set.Set("storage.notification.clean_interval", "0")
			Expect(err).ToNot(HaveOccurred())
		}).Build()
	})

	AfterSuite(func() {
		testContext.Cleanup()
	})

	Context("When two notifications are inserted", func() {
		It("Should delete the old one", func() {
			idNew, err := repository.Create(ctx, randomNotification())
			Expect(err).ToNot(HaveOccurred())
			Expect(idNew).ToNot(BeEmpty())

			oldNotification := randomNotification()
			oldNotification.CreatedAt = time.Now().Add(-defaultOlderThan)
			idOld, err := repository.Create(ctx, oldNotification)
			Expect(err).ToNot(HaveOccurred())
			Expect(idOld).ToNot(BeNil())

			obj, err := repository.Get(ctx, types.NotificationType, idNew)
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.GetID()).To(Equal(idNew))

			// TODO remove sleep and use interceptor channel
			time.Sleep(time.Second)
			_, err = repository.Get(ctx, types.NotificationType, idOld)
			Expect(err).To(Equal(util.ErrNotFoundInStorage))
		})
	})
})
