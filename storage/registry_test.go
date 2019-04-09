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
	"testing"
	"time"

	"sync"

	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Suite")
}

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

	BeforeEach(func() {
		testStorage = &storagefakes.FakeStorage{}
		testStorage.OpenReturns(nil)
		testStorage.CloseReturns(nil)
		testSettings = &storage.Settings{
			URI:           "uri",
			MigrationsURL: "",
			EncryptionKey: "",
		}
	})

	Describe("Storage registration", func() {
		var (
			name string
			s    storage.Storage
		)

		registerStorage := func() {
			storage.Register(name, s)
		}

		assertStorageRegistrationPanics := func() {
			Expect(registerStorage).To(Panic())
		}

		Context("With nil storage", func() {
			It("Should panic", func() {
				name = "storage"
				assertStorageRegistrationPanics()
			})
		})

		Context("With duplicate storage name", func() {
			It("Should panic", func() {
				name = "duplicate"
				s = testStorage
				registerStorage()
				assertStorageRegistrationPanics()
			})
		})
	})

	Describe("Use storage", func() {
		Context("With non-registered", func() {
			It("Should return an error", func() {
				returnedStorage, err := storage.Use(context.TODO(), "non-existing-storage", &storage.Settings{
					URI:           "uri",
					MigrationsURL: "",
					EncryptionKey: "",
				})
				Expect(returnedStorage).To(BeNil())
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("When opening fails", func() {
			It("Should return an error", func() {
				testStorage.OpenReturns(fmt.Errorf("Error"))
				storage.Register("openFailingStorage", testStorage)
				_, err := storage.Use(context.TODO(), "openFailingStorage", testSettings)
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("When opening succeeds", func() {
			It("Should return storage", func() {
				testStorage.OpenReturns(nil)
				storage.Register("openOkStorage", testStorage)
				configuredStorage, err := storage.Use(context.TODO(), "openOkStorage", testSettings)
				Expect(configuredStorage).To(Not(BeNil()))
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("Close storage", func() {
		var interceptor *logInterceptor

		BeforeEach(func() {
			interceptor = &logInterceptor{}
			logrus.AddHook(interceptor)
		})

		Context("When close fails", func() {
			It("Should panic", func() {
				testStorage.CloseReturns(fmt.Errorf("Error"))
				storage.Register("closeFailingStorage", testStorage)
				ctx, cancel := context.WithCancel(context.TODO())
				storage.Use(ctx, "closeFailingStorage", testSettings)
				cancel()
				time.Sleep(time.Millisecond * 100)
				interceptor.VerifyData(false)
			})
		})

		Context("When close succeeds", func() {
			It("Should be ok", func() {
				testStorage.CloseReturns(nil)
				storage.Register("closeOkStorage", testStorage)
				ctx, cancel := context.WithCancel(context.TODO())
				storage.Use(ctx, "closeOkStorage", testSettings)
				cancel()
				time.Sleep(time.Millisecond * 100)
				interceptor.VerifyData(true)
			})
		})
	})
})
