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
	"fmt"
	"sync"

	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type logInterceptor struct {
	data string
	lock sync.RWMutex
}

func (*logInterceptor) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (interceptor *logInterceptor) Fire(e *logrus.Entry) (err error) {
	interceptor.lock.Lock()
	defer interceptor.lock.Unlock()

	interceptor.data, err = e.String()
	return
}

func (interceptor *logInterceptor) VerifyData(emptyData bool) {
	interceptor.lock.Lock()
	defer interceptor.lock.Unlock()

	if emptyData {
		Eventually(interceptor.data).Should(BeEmpty())
	} else {
		Eventually(interceptor.data).Should(Not(BeEmpty()))
	}
}

var _ = Describe("Registry", func() {
	var testStorage *storagefakes.FakeStorage
	var testSettings *storage.Settings
	var ctx context.Context
	var cancelFunc func()
	var wg *sync.WaitGroup

	BeforeEach(func() {
		testStorage = &storagefakes.FakeStorage{}
		testSettings = &storage.Settings{
			URI:           "uri",
			MigrationsURL: "migrationuri",
			EncryptionKey: "encryptionkey",
		}

		ctx, cancelFunc = context.WithCancel(context.TODO())
		wg = &sync.WaitGroup{}
	})

	Describe("InitializeWithSafeTermination", func() {
		Context("when storage is nil", func() {
			BeforeEach(func() {
				testStorage = nil
			})

			It("panics", func() {
				Expect(func() {
					storage.InitializeWithSafeTermination(ctx, testStorage, testSettings, wg)
				}).To(Panic())
			})
		})

		Context("when opening the storage returns an error", func() {
			BeforeEach(func() {
				testStorage.OpenReturns(fmt.Errorf("error"))
			})

			It("returns an error", func() {
				_, err := storage.InitializeWithSafeTermination(ctx, testStorage, testSettings, wg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when context is cancelled", func() {
			Context("when close succeeds", func() {
				It("it closes the storage when the context is canceled", func() {
					_, err := storage.InitializeWithSafeTermination(ctx, testStorage, testSettings, wg)
					Expect(err).ToNot(HaveOccurred())

					Expect(testStorage.CloseCallCount()).To(Equal(0))

					cancelFunc()
					wg.Wait()

					Expect(testStorage.CloseCallCount()).To(Equal(1))
				})
			})

			Context("when close returns an error", func() {
				var logHook *logInterceptor

				BeforeEach(func() {
					logHook = &logInterceptor{}
					testStorage.CloseReturns(fmt.Errorf("error"))
					logrus.AddHook(logHook)
				})

				It("logs the error", func() {
					_, err := storage.InitializeWithSafeTermination(ctx, testStorage, testSettings, wg)
					Expect(err).ToNot(HaveOccurred())

					cancelFunc()
					wg.Wait()

					logHook.VerifyData(false)
				})
			})
		})

		Context("when no decorators are provided", func() {
			It("returns the storage that was supplied as argument", func() {
				s, err := storage.InitializeWithSafeTermination(ctx, testStorage, testSettings, wg)
				Expect(err).ToNot(HaveOccurred())
				Expect(s).To(Equal(testStorage))
			})
		})

		Context("when decorators are provided", func() {
			It("wraps the specified storage in the decorators", func() {
				invocations := 0
				_, err := storage.InitializeWithSafeTermination(ctx, testStorage, testSettings, wg, func(transactionalRepository storage.TransactionalRepository) (storage.TransactionalRepository, error) {
					invocations++
					return transactionalRepository, nil
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(invocations).To(Equal(1))
			})
		})
	})
})
