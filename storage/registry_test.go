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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"github.com/Peripli/service-manager/storage"
	"context"
	"fmt"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/sirupsen/logrus"
	"time"
)

func TestStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Suite")
}

type logInterceptor struct {
	data string
}

func (*logInterceptor) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (interceptor *logInterceptor) Fire(e *logrus.Entry) (err error) {
	interceptor.data, err = e.String()
	return
}

var _ = Describe("Registry", func() {
	var testStorage *storagefakes.FakeStorage

	BeforeEach(func() {
		testStorage = &storagefakes.FakeStorage{}
		testStorage.OpenReturns(nil)
		testStorage.CloseReturns(nil)
	})

	Describe("Storage registration", func() {
		Context("With nil storage", func() {
			It("Should panic", func() {
				registerNilStorage := func() {
					storage.Register("storage", nil)
				}
				Expect(registerNilStorage).To(Panic())
			})
		})

		Context("With duplicate storage name", func() {
			It("Should panic", func() {
				registerStorage := func() {
					storage.Register("duplicate", testStorage)
				}
				registerStorage()
				Expect(registerStorage).To(Panic())
			})
		})
	})

	Describe("Use storage", func() {
		Context("With non-registered", func() {
			It("Should return an error", func() {
				returnedStorage, err := storage.Use("non-existing-storage", "uri", context.TODO())
				Expect(returnedStorage).To(BeNil())
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("When opening fails", func() {
			It("Should return an error", func() {
				testStorage.OpenReturns(fmt.Errorf("Error"))
				storage.Register("openFailingStorage", testStorage)
				_, err := storage.Use("openFailingStorage", "uri", context.TODO())
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("When opening succeeds", func() {
			It("Should return storage", func() {
				testStorage.OpenReturns(nil)
				storage.Register("openOkStorage", testStorage)
				configuredStorage, err := storage.Use("openOkStorage", "uri", context.TODO())
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
				storage.Use("closeFailingStorage", "uri", ctx)
				cancel()
				time.Sleep(time.Millisecond * 100)
				Expect(interceptor.data).To(Not(BeEmpty()))
			})
		})

		Context("When close succeeds", func() {
			It("Should be ok", func() {
				testStorage.CloseReturns(nil)
				storage.Register("closeOkStorage", testStorage)
				ctx, cancel := context.WithCancel(context.TODO())
				storage.Use("closeOkStorage", "uri", ctx)
				cancel()
				time.Sleep(time.Millisecond * 100)
				Eventually(interceptor.data).Should(BeEmpty())
			})
		})
	})
})
