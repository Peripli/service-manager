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

	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/types"
)

// BrokerDetailsFetcher implements osb.BrokerFetchervalidatable
type BrokerDetailsFetcher struct {
	Username string
	Password string
	URL      string
}

var _ osb.BrokerFetcher = &BrokerDetailsFetcher{}

// FetchBroker implements osb.BrokerRoundTripper and returns the coordinates of the broker with the specified id
func (b *BrokerDetailsFetcher) FetchBroker(ctx context.Context, brokerID string) (*types.Broker, error) {
	return &types.Broker{
		BrokerURL: b.URL + "/" + brokerID,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: b.Username,
				Password: b.Password,
			},
		},
	}, nil
}
