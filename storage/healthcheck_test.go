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

package storage_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthcheck", func() {
	var healthIndicator health.Indicator

	BeforeEach(func() {
		ping := func(ctx context.Context) error {
			return nil
		}
		var err error
		healthIndicator, err = storage.NewSQLHealthIndicator(storage.PingFunc(ping))
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Name", func() {
		It("Returns text", func() {
			Expect(healthIndicator.Name()).ToNot(BeEmpty())
		})
	})

	Context("Ping does not return error", func() {
		It("Status doest not contains error", func() {
			_, err := healthIndicator.Status()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
	Context("Ping returns error", func() {
		expectedError := fmt.Errorf("could not connect to database")
		BeforeEach(func() {
			ping := func(ctx context.Context) error {
				return expectedError
			}
			var err error
			healthIndicator, err = storage.NewSQLHealthIndicator(storage.PingFunc(ping))
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Contains error", func() {
			_, err := healthIndicator.Status()
			Expect(err).Should(HaveOccurred())
			Expect(err).To(Equal(expectedError))
		})
	})
})
