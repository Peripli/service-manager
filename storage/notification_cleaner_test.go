/*
 * Copyright 2018 The Service Manager Authors
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

package storage_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/storagefakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notification cleaner", func() {
	var (
		ctx         context.Context
		cancel      context.CancelFunc
		wg          *sync.WaitGroup
		fakeStorage *storagefakes.FakeStorage
		nc          *storage.NotificationCleaner
	)

	BeforeEach(func() {
		fakeStorage = &storagefakes.FakeStorage{}
		ctx, cancel = context.WithCancel(context.Background())
		wg = &sync.WaitGroup{}
		nc = &storage.NotificationCleaner{
			Storage:  fakeStorage,
			Settings: *storage.DefaultSettings(),
		}
	})

	AfterEach(func() {
		if ctx.Err() == nil {
			cancel()
		}
		wg.Wait()
	})

	Describe("Start", func() {

		Context("When already started", func() {
			It("Should return error", func() {
				err := nc.Start(ctx, wg)
				Expect(err).ToNot(HaveOccurred())
				err = nc.Start(ctx, wg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("notification cleaner already started"))
			})
		})

	})

	Describe("clean", func() {
		Context("When scheduled", func() {
			It("Should call storage.DeleteReturning", func() {
				nc.Settings.Notification.CleanInterval = 0
				var objType types.ObjectType
				var criteria []query.Criterion
				fakeStorage.DeleteStub = func(ctx context.Context, objectType types.ObjectType, criterion ...query.Criterion) error {
					objType = objectType
					criteria = criterion
					cancel() // stop notification cleaner
					return nil
				}
				err := nc.Start(ctx, wg)
				Expect(err).ToNot(HaveOccurred())
				wg.Wait()
				Expect(objType).To(Equal(types.NotificationType))
				Expect(criteria).To(HaveLen(1))
				Expect(criteria[0].LeftOp).To(Equal("created_at"))
				timeString := criteria[0].RightOp[0]
				timeQueryParameter, err := time.Parse(time.RFC3339, timeString)
				Expect(err).ToNot(HaveOccurred())
				Expect(timeQueryParameter).To(BeTemporally("<", time.Now()))
			})
		})

		checkCleanerNotStopped := func(storageError error) {
			nc.Settings.Notification.CleanInterval = 0
			called := false
			fakeStorage.DeleteStub = func(ctx context.Context, objectType types.ObjectType, criterion ...query.Criterion) error {
				if !called {
					called = true
					return storageError
				} else {
					cancel()
					return nil
				}
			}
			err := nc.Start(ctx, wg)
			Expect(err).ToNot(HaveOccurred())
			wg.Wait()
			Expect(called).To(BeTrue())
		}

		Context("When repository returns not found error", func() {
			It("Should not stop", func() {
				checkCleanerNotStopped(util.ErrNotFoundInStorage)
			})
		})

		Context("When repository returns error", func() {
			It("Should not stop", func() {
				checkCleanerNotStopped(errors.New("*Expected*"))
			})
		})
	})
})
