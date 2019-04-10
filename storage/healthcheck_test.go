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
	"fmt"

	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthcheck", func() {
	var healthIndicator *storage.HealthIndicator
	var pinger *storagefakes.FakePinger

	BeforeEach(func() {
		pinger = &storagefakes.FakePinger{}
		pinger.PingStub = func() error {
			return nil
		}
		healthIndicator = &storage.HealthIndicator{
			Pinger: pinger,
		}
	})

	Context("Name", func() {
		It("Returns text", func() {
			Expect(healthIndicator.Name()).ToNot(BeEmpty())
		})
	})

	Context("Ping does not return error", func() {
		It("Returns health status UP", func() {
			healthz := healthIndicator.Health()
			Expect(healthz.Status).To(Equal(health.StatusUp))
		})
	})
	Context("Ping returns error", func() {
		expectedError := fmt.Errorf("could not connect to database")
		BeforeEach(func() {
			pinger.PingStub = func() error {
				return expectedError
			}
		})
		It("Returns status DOWN", func() {
			healthz := healthIndicator.Health()
			Expect(healthz.Status).To(Equal(health.StatusDown))
		})
		It("Contains error", func() {
			healthz := healthIndicator.Health()
			errorDetails := healthz.Details["error"]
			Expect(errorDetails).To(Equal(expectedError))
		})

	})
})
