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

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
)

// StorageBrokerFetcher provides logic for fetching the broker coordinates from the storage
type StorageBrokerFetcher struct {
	BrokerStorage storage.Broker
	Encrypter     security.Encrypter
}

var _ BrokerFetcher = &StorageBrokerFetcher{}

// FetchBroker obtains the broker coordinates (auth and URL)
func (sbf *StorageBrokerFetcher) FetchBroker(ctx context.Context, brokerID string) (*types.Broker, error) {
	broker, err := sbf.BrokerStorage.Get(ctx, brokerID)
	if err != nil {
		log.C(ctx).Debugf("FetchBroker with id %s not found in storage", brokerID)
		return nil, util.HandleStorageError(err, "broker")
	}

	password := broker.Credentials.Basic.Password
	plaintextPassword, err := sbf.Encrypter.Decrypt(ctx, []byte(password))
	if err != nil {
		return nil, err
	}

	broker.Credentials.Basic.Password = string(plaintextPassword)

	return broker, nil
}
