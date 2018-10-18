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

package platform

import "context"

// Client provides the logic for calling into the underlying platform and performing platform specific operations
//go:generate counterfeiter . Client
type Client interface {
	// GetBrokers obtains the registered brokers in the platform
	GetBrokers(ctx context.Context) ([]ServiceBroker, error)

	// CreateBroker registers a new broker at the platform
	CreateBroker(ctx context.Context, r *CreateServiceBrokerRequest) (*ServiceBroker, error)

	// DeleteBroker unregisters a broker from the platform
	DeleteBroker(ctx context.Context, r *DeleteServiceBrokerRequest) error

	// UpdateBroker updates a broker registration at the platform
	UpdateBroker(ctx context.Context, r *UpdateServiceBrokerRequest) (*ServiceBroker, error)
}
