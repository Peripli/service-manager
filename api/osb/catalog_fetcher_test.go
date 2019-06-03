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
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/api/osb"

	"github.com/Peripli/service-manager/pkg/util"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
)

var _ = Describe("Catalog CatalogFetcher", func() {
	const (
		simpleCatalog = `
		{
		  "services": [{
				"name": "no-tags-no-metadata",
				"id": "acb56d7c-XXXX-XXXX-XXXX-feb140a59a67",
				"description": "A fake service.",
				"dashboard_client": {
					"id": "id",
					"secret": "secret",
					"redirect_uri": "redirect_uri"		
				},    
				"plans": [{
					"random_extension": "random_extension",
					"name": "fake-plan-1",
					"id": "d3031751-XXXX-XXXX-XXXX-a42377d33202",
					"description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections.",
					"free": false
				}]
			}]
		}`

		id       = "id"
		name     = "name"
		url      = "url"
		username = "username"
		password = "password"
		version  = "2.13"
	)

	var testBroker *types.ServiceBroker
	var expectedHeaders map[string]string

	type testCase struct {
		expectations *common.HTTPExpectations
		reaction     *common.HTTPReaction

		expectedErr      error
		expectedResponse []byte
	}

	newFetcher := func(t testCase) func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error) {
		return osb.CatalogFetcher(common.DoHTTP(t.reaction, t.expectations), version)
	}

	basicAuth := func(username, password string) string {
		auth := username + ":" + password
		return base64.StdEncoding.EncodeToString([]byte(auth))
	}

	BeforeEach(func() {
		testBroker = &types.ServiceBroker{
			Base: types.Base{
				ID:     id,
				Labels: map[string][]string{},
			},
			Name:      name,
			BrokerURL: url,
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: username,
					Password: password,
				},
			},
		}

		expectedHeaders = map[string]string{
			"Authorization":        "Basic " + basicAuth(username, password),
			"X-Broker-API-Version": version,
		}
	})

	entries := []TableEntry{
		Entry("successfully fetches the catalog bytes", testCase{
			expectations: &common.HTTPExpectations{
				URL:     url,
				Headers: expectedHeaders,
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
				Body:   simpleCatalog,
				Err:    nil,
			},
			expectedErr:      nil,
			expectedResponse: []byte(simpleCatalog),
		}),
		Entry("returns error if response code from broker is not 200", testCase{
			expectations: &common.HTTPExpectations{
				URL:     url,
				Headers: expectedHeaders,
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusInternalServerError,
				Body:   simpleCatalog,
				Err:    nil,
			},
			expectedErr:      fmt.Errorf("error fetching catalog for broker with name %s", name),
			expectedResponse: nil,
		}),
		Entry("returns error if sending request fails with error", testCase{
			expectations: &common.HTTPExpectations{
				URL:     url,
				Headers: expectedHeaders,
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusBadGateway,
				Err:    fmt.Errorf("error sending request"),
			},
			expectedErr: &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("could not reach service broker %s at %s", name, url),
				StatusCode:  http.StatusBadGateway,
			},
		}),
	}

	DescribeTable("Fetch", func(t testCase) {
		fetcher := newFetcher(t)
		rawCatalog, err := fetcher(context.TODO(), testBroker)

		if t.expectedErr != nil {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(t.expectedErr.Error()))
		} else {
			Expect(err).ToNot(HaveOccurred())
		}

		if t.expectedResponse != nil {
			Expect(rawCatalog).To(Equal(t.expectedResponse))
		}
	}, entries...)
})
