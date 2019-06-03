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

package osb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
)

const brokerCatalogURL = "%s/v2/catalog"
const brokerAPIVersionHeader = "X-Broker-API-Version"

// CatalogFetcher creates a broker catalog fetcher that uses the provided request function to call the specified broker's catalog endpoint
func CatalogFetcher(doRequestFunc util.DoRequestFunc, brokerAPIVersion string) func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error) {
	return func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error) {
		log.C(ctx).Debugf("Attempting to fetch catalog from broker with name %s and URL %s", broker.Name, broker.BrokerURL)
		requestWithBasicAuth := util.BasicAuthDecorator(broker.Credentials.Basic.Username, broker.Credentials.Basic.Password, doRequestFunc)
		response, err := util.SendRequestWithHeaders(ctx, requestWithBasicAuth, http.MethodGet, fmt.Sprintf(brokerCatalogURL, broker.BrokerURL), map[string]string{}, nil, map[string]string{
			brokerAPIVersionHeader: brokerAPIVersion,
		})
		if err != nil {
			log.C(ctx).WithError(err).Errorf("Error while forwarding request to service broker %s", broker.Name)
			return nil, &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("could not reach service broker %s at %s", broker.Name, broker.BrokerURL),
				StatusCode:  http.StatusBadGateway,
			}
		}

		var responseBytes []byte
		if responseBytes, err = util.BodyToBytes(response.Body); err != nil {
			return nil, fmt.Errorf("error getting content from body of response with status %s: %s", response.Status, err)
		}

		if response.StatusCode != http.StatusOK {
			log.C(ctx).WithError(err).Errorf("error fetching catalog for broker with name %s: %s", broker.Name, util.HandleResponseError(response))
			return nil, &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("error fetching catalog for broker with name %s: broker responded with %s", broker.Name, response.Status),
				StatusCode:  http.StatusBadRequest,
			}
		}
		log.C(ctx).Debugf("Successfully fetched catalog from broker with name %s and URL %s", broker.Name, broker.BrokerURL)

		return responseBytes, nil
	}
}
