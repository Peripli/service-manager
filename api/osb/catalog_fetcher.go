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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

const brokerCatalogURL = "%s/v2/catalog"
const brokerAPIVersionHeader = "X-Broker-API-Version"

// CatalogFetcher creates a broker catalog fetcher that uses the provided request function to call the specified broker's catalog endpoint
func CatalogFetcher(doRequestWithClient util.DoRequestWithClientFunc, brokerAPIVersion string) func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error) {
	return func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error) {
		return Get(doRequestWithClient, brokerAPIVersion, ctx, broker, fmt.Sprintf(brokerCatalogURL, broker.BrokerURL), "catalog")
	}
}
