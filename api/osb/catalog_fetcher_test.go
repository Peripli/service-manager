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
		tlsKey   = `-----BEGIN PRIVATE KEY-----
  MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDaO3W1LP5M20sF
  fnPI+s3pqVRPbnHe5TiepguuMLqcM4HS6eJz5/IimmILUexCLZ83WZYOcAGFqRNR
  zLUrhbOH62RK+U8JvaB4JA/rFzXOQ698RDzAVo7ZFhiHGO3o1Y27icdfF2ps2MZX
  CY6UxK1x7P1ZYXds4gefJQaqiZrIcuwfb97+hIlgVwYh6k3AkBQMqL0gb/vZhH+I
  2BdHZCIDDTbenSwegP0IIyneg75IQDVQJydzR4i5JswXclgofpLi5A7s/WXCIhg/
  B5ODlKpbU+ziV1lrSnmEPtPU6UKh9iJGaukHbJMHFo0z3B0MfGOUxpPEaV3n6Rpg
  Zp8Kkw9tAgMBAAECggEADmKc/7RXjvllmJcdSsI9kIl45UOCfg7eDJclbfYIVwOO
  Kzj/lGRVsbI7hEOCL1qShDODkLARaZ4bh+jWiGfnza3WjpqgeyPk0AaQhg6hnVcY
  2jglSQhroiOyujUKea6aCSKr4bjJayNe753RqDzOshPNH3ctSCAeIH9wUQ2BBnVt
  r+kbDbrXr834/XkOB73r85CX+d690THYu1CdqT6OAju74u+19gC1UhCn95F/Sa/v
  ej0TEC8EqUOcsRpfjUEDx6Ywwr5RrblzEaS997IX0LZb21/8g6qVUbq+oKZtvyOe
  P/cq17cuvr3pvKhrGi5uOlX1JPan969jQqe7xxJkgQKBgQDxsRHAP4y1tgjMaaoM
  dkFA4WoQ+ilevIVtfxO5beb89ooANcjlX7lnGcAL1WAjoa5lr05No2pHudE1ivdt
  eJgER13/BtgVjz65E5gzYEjDqcle2K/3LVIOslvFk0M9umflZdjlz0fl1IlutcMd
  4lZ3cS8zJPVCR+D2CyDH+IxDuwKBgQDnJtqgPLjtlRxmjXaQ4zMcFpcdTqXr0Ddf
  AaDA0iC+mihNfmxmpMjPmpiROO8MtrvvQy9Kzdg53ptl6e2alzcWoJ8p6vDnMFUa
  PuIf3/lw9PfLkImGDiRyn6GcwGaWluHUXlphJvTi5A2Ql5HmW9XHlwRx/ZGur7bm
  lRhsAB7C9wKBgQDgzC0SfwlFSdbNKcp8ZNE0o3Sf7c3ky7veqD+UTOB3kGey4lPE
  5E/x0UWKvB/7hDpNYcyW8dO8etxXzLVuIKhj8m0+8wKwqtdQFSWPQ5LqSlV93lVs
  tb6I5OPu1JXKKELSXvRqa20YG6LoUi708LwzxBZ+n3Vu/KQEtTz8QfVUWQKBgQCA
  hZrzky+jcc/7uVYeUyU8zdaxxeP9TKUs3wPZkjwAnkggZlWxcJfyzltcC5Lmt8eg
  zfNCnVdHPd2becjRtpg7rY0xyl6tvLLkx+gEnwzbYGlStwewELb1QIqkVFn2Cuh/
  owKPmBB7AyADsDLAKXmg4vfmxX016p9Ab8/HZP21mwKBgC8MRd0XwV3sCR1PmW3O
  vrlZ+SswoN/7pwRTRWX/S0AHjYBJ+Bn25p3v4R1PcaESpYzYnuDWAgIl2uqncdeA
  IlSzIMiKfxuJIOzpJHEdwhVmFriEIIrwdAA0jPMLeXGFTmI/vWCmcG9F5XGwlRRJ
  gB9ceWwBvf0HhZYfJ3XCJZXM
-----END PRIVATE KEY-----`

		cert = `-----BEGIN CERTIFICATE-----
MIICrDCCAZQCCQCziU7at44ipjANBgkqhkiG9w0BAQUFADAYMRYwFAYDVQQDDA1G
aXJzdCBNLiBMYXN0MB4XDTIwMDIyMjEyNDMxMloXDTIwMDMyMzEyNDMxMlowGDEW
MBQGA1UEAwwNRmlyc3QgTS4gTGFzdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBANo7dbUs/kzbSwV+c8j6zempVE9ucd7lOJ6mC64wupwzgdLp4nPn8iKa
YgtR7EItnzdZlg5wAYWpE1HMtSuFs4frZEr5Twm9oHgkD+sXNc5Dr3xEPMBWjtkW
GIcY7ejVjbuJx18XamzYxlcJjpTErXHs/Vlhd2ziB58lBqqJmshy7B9v3v6EiWBX
BiHqTcCQFAyovSBv+9mEf4jYF0dkIgMNNt6dLB6A/QgjKd6DvkhANVAnJ3NHiLkm
zBdyWCh+kuLkDuz9ZcIiGD8Hk4OUqltT7OJXWWtKeYQ+09TpQqH2IkZq6QdskwcW
jTPcHQx8Y5TGk8RpXefpGmBmnwqTD20CAwEAATANBgkqhkiG9w0BAQUFAAOCAQEA
QcLPaEwZ6EYoY7aa4sOzkV4AENEkdLcz/DQOFns5LisFtCUbGoPufzs4ozn9Bngy
fTSrUqV/I5l7bQV18vhWH86OqBYiDrxZMaTIgySuzN3aXJCpsw4JP0rHZjrRjFDx
hpL8qoDDR9vDvjvqE2jlqXMPAe0DZEljRzG+EARODnaCEFFpzEkosQLlPSXyn51I
3ffwNHcPQQeZCknqJ9BI8a4JdEP1cZDdl6TPu1rsakFfCHSKCwrKa6blCZRxVvpd
qYxHGtKZSU5BCswd7c3r8SL5qzmAscmu6orqwzGsvLHAx3Y9OcF+7weDZdz2OB3p
OOzY8kGVInUs83tZOfMVjQ==
-----END CERTIFICATE-----`
	)

	var expectedHeaders map[string]string

	testBroker := types.ServiceBroker{
		Base: types.Base{
			ID:     id,
			Labels: map[string][]string{},
			Ready:  true,
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

	testBrokeTLS := types.ServiceBroker{
		Base: types.Base{
			ID:     id,
			Labels: map[string][]string{},
			Ready:  true,
		},
		Name:      name,
		BrokerURL: url,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: username,
				Password: password,
			},
			TLS: &types.TLS{
				Certificate: cert,
				Key:         tlsKey,
			},
		},
	}

	testBrokerTLSInvalid := types.ServiceBroker{
		Base: types.Base{
			ID:     id,
			Labels: map[string][]string{},
			Ready:  true,
		},
		Name:      name,
		BrokerURL: url,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: username,
				Password: password,
			},
			TLS: &types.TLS{
				Certificate: "cert",
				Key:         "key",
			},
		},
	}

	type testCase struct {
		expectations     *common.HTTPExpectations
		reaction         *common.HTTPReaction
		broker           types.ServiceBroker
		expectedErr      error
		expectedResponse []byte
	}

	newFetcher := func(t testCase) func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error) {
		return osb.CatalogFetcher(common.DoHTTPWithClient(t.reaction, t.expectations), version)
	}

	basicAuth := func(username, password string) string {
		auth := username + ":" + password
		return base64.StdEncoding.EncodeToString([]byte(auth))
	}

	BeforeEach(func() {
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
			broker:           testBroker,
		}),
		Entry("returns error if response code from broker is not 200", testCase{
			broker: testBroker,
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
			broker: testBroker,
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
		Entry("returns error if invalid tls settings are passed", testCase{
			broker: testBrokerTLSInvalid,
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
				Description: fmt.Sprintf("failed to find any PEM data in certificate input"),
				StatusCode:  http.StatusBadGateway,
			},
		}),
		Entry("returns error if invalid tls settings are passed", testCase{
			broker: testBrokeTLS,
			expectations: &common.HTTPExpectations{
				URL:     url,
				Headers: expectedHeaders,
			},
			reaction: &common.HTTPReaction{
				Status: http.StatusOK,
			},
			expectedErr: nil,
		}),
	}

	DescribeTable("Fetch", func(t testCase) {
		fetcher := newFetcher(t)
		rawCatalog, err := fetcher(context.TODO(), &t.broker)

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
