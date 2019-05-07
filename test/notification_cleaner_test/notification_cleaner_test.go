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
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
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
	var deleteInterceptor *storagefakes.FakeDeleteInterceptor
	var deleteInterceptorProvider *storagefakes.FakeDeleteInterceptorProvider
	var continueDelete chan struct{}
	var deleteFinished chan struct{}

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

	getNotification := func(id string) {
		obj, err := repository.Get(ctx, types.NotificationType, id)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj.GetID()).To(Equal(id))
	}

	BeforeSuite(func() {
		ctx = context.Background()
		deleteInterceptor = &storagefakes.FakeDeleteInterceptor{}
		deleteInterceptor.OnTxDeleteStub = func(h storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
			return h
		}
		continueDelete = make(chan struct{})
		deleteFinished = make(chan struct{})
		deleteInterceptor.AroundTxDeleteStub = func(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
			return func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
				select {
				case <-continueDelete:
					break
				case <-ctx.Done():
					return nil, nil
				}
				defer func() {
					select {
					case deleteFinished <- struct{}{}:
						break
					case <-ctx.Done():
						break
					}
				}()
				return h(ctx, deletionCriteria...)
			}
		}
		deleteInterceptorProvider = &storagefakes.FakeDeleteInterceptorProvider{}
		deleteInterceptorProvider.NameReturns("tetInterceptor")
		deleteInterceptorProvider.ProvideReturns(deleteInterceptor)
		defaultOlderThan = storage.DefaultSettings().Notification.OlderThan
		testContext = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			repository = smb.Storage
			smb.WithDeleteInterceptorProvider(types.NotificationType, deleteInterceptorProvider).Register()
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
			oldNotification.CreatedAt = time.Now().Add(-defaultOlderThan - time.Hour)
			idOld, err := repository.Create(ctx, oldNotification)
			Expect(err).ToNot(HaveOccurred())
			Expect(idOld).ToNot(BeNil())

			getNotification(idOld)
			getNotification(idNew)

			continueDelete <- struct{}{}
			<-deleteFinished
			_, err = repository.Get(ctx, types.NotificationType, idOld)
			Expect(err).To(Equal(util.ErrNotFoundInStorage))
			getNotification(idNew)
		})
	})
})
