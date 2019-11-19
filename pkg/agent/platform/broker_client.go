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

import (
	"context"
)

// CreateServiceBrokerRequest type used for requests by the platform client
type CreateServiceBrokerRequest struct {
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

// UpdateServiceBrokerRequest type used for requests by the platform client
type UpdateServiceBrokerRequest struct {
	GUID      string `json:"guid"`
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

// DeleteServiceBrokerRequest type used for requests by the platform client
type DeleteServiceBrokerRequest struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

// ServiceBroker type for responses from the platform client
type ServiceBroker struct {
	GUID      string `json:"guid"`
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

// ServiceBrokerList type for responses from the platform client
type ServiceBrokerList struct {
	ServiceBrokers []ServiceBroker `json:"service_brokers"`
}

// BrokerClient provides the logic for calling into the underlying platform and performing platform specific operations
//go:generate counterfeiter . BrokerClient
type BrokerClient interface {
	// GetBrokers obtains the registered brokers in the platform
	GetBrokers(ctx context.Context) ([]*ServiceBroker, error)

	// GetBrokerByName returns the broker from the platform with the specified name
	GetBrokerByName(ctx context.Context, name string) (*ServiceBroker, error)

	// CreateBroker registers a new broker at the platform
	CreateBroker(ctx context.Context, r *CreateServiceBrokerRequest) (*ServiceBroker, error)

	// DeleteBroker unregisters a broker from the platform
	DeleteBroker(ctx context.Context, r *DeleteServiceBrokerRequest) error

	// UpdateBroker updates a broker registration at the platform
	UpdateBroker(ctx context.Context, r *UpdateServiceBrokerRequest) (*ServiceBroker, error)
}
