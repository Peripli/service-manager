/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package storage

import "github.com/Peripli/service-manager/storage/dto"

// Provider interface for db storage provider
type Provider interface {
	Provide() (Storage, error)
}

// Storage interface for db storage
type Storage interface {
	Broker() Broker
	Platform() Platform
	Credentials() Credentials
}

// Broker interface for Broker db operations
type Broker interface {
	Create(broker *dto.Broker) error
	Get(id string) (*dto.Broker, error)
	GetAll() ([]*dto.Broker, error)
	Delete(id string) error
	Update(broker *dto.Broker) error
}

// Platform interface for Platform db operations
type Platform interface {
	Create(platform *dto.Platform) error
	Get(id string) (*dto.Platform, error)
	GetAll() ([]*dto.Platform, error)
	Delete(id string) error
	Update(platform *dto.Platform) error
}

// Credentials interface for Credentials db operations
type Credentials interface {
	Create(credentials *dto.Credentials) (int, error)
}
