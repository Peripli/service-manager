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

package app

import (
	"context"

	"github.com/Peripli/service-manager/pkg/sbproxy/platform"
)

var _ platform.CatalogFetcher = &PlatformClient{}

// Fetch implements service-broker-proxy/pkg/cf/Fetcher.Fetch and provides logic for triggering refetching
// of the broker's catalog
func (pc PlatformClient) Fetch(ctx context.Context, broker *platform.ServiceBroker) error {
	_, err := pc.UpdateBroker(ctx, &platform.UpdateServiceBrokerRequest{
		GUID:      broker.GUID,
		Name:      broker.Name,
		BrokerURL: broker.BrokerURL,
	})

	return err
}
