/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package postgres

import (
	"context"
	"testing"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/security"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal Security Storage Test Suite")
}

var _ = Describe("Security Storage", func() {
	var securityStorage security.Storage

	BeforeEach(func() {
		securityStorage = &storage{}
	})

	Context("New Security Storage", func() {
		Context("With empty URI", func() {
			It("Should return error", func() {
				secureStorage, err := NewSecureStorage(context.TODO(), api.Security{})
				Expect(secureStorage).To(BeNil())
				Expect(err).To(Not(BeNil()))
			})
		})
	})

	Context("Fetcher", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() { securityStorage.Fetcher() }).To(Panic())
			})
		})
	})

	Context("Setter", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() { securityStorage.Setter() }).To(Panic())
			})
		})
	})
})
