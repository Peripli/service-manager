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

package osb_test

import (
	"context"

	smosb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/sbproxy/osb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BrokerFetcher", func() {
	Describe("FetcherBroker", func() {
		const (
			user     = "admin"
			password = "admin"
			url      = "https://example.com"
			brokerID = "brokerID"
		)
		var fetcher smosb.BrokerFetcher

		BeforeEach(func() {
			fetcher = &osb.BrokerDetailsFetcher{
				Username: user,
				Password: password,
				URL:      url,
			}
		})

		It("returns a broker type with correct broker details", func() {
			broker, err := fetcher.FetchBroker(context.TODO(), brokerID)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(broker.Credentials.Basic.Username).To(Equal(user))
			Expect(broker.Credentials.Basic.Password).To(Equal(password))
			Expect(broker.BrokerURL).To(Equal(url + "/" + brokerID))
		})
	})
})
