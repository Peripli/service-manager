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

package interceptors

import (
	"time"

	"github.com/Peripli/service-manager/storage"
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
)

const ServiceBindingCreateInterceptorProviderName = "ServiceBindingCreateInterceptorProvider"

// ServiceBindingCreateInterceptorProvider provides an interceptor that notifies the actual broker about instance creation
type ServiceBindingCreateInterceptorProvider struct {
	OSBClientCreateFunc  osbc.CreateFunc
	Repository           storage.TransactionalRepository
	TenantKey            string
	PollingInterval      time.Duration
	MaxParallelDeletions int
}

func (p *ServiceBindingCreateInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &ServiceBindingInterceptor{
		osbClientCreateFunc:  p.OSBClientCreateFunc,
		repository:           p.Repository,
		tenantKey:            p.TenantKey,
		pollingInterval:      p.PollingInterval,
		maxParallelDeletions: p.MaxParallelDeletions,
	}
}

func (c *ServiceBindingCreateInterceptorProvider) Name() string {
	return ServiceBindingCreateInterceptorProviderName
}

const ServiceBindingUpdateInterceptorProviderName = "ServiceBindingUpdateInterceptorProvider"

// ServiceBindingUpdateInterceptorProvider provides an interceptor that notifies the actual broker about instance updates
type ServiceBindingUpdateInterceptorProvider struct {
	OSBClientCreateFunc  osbc.CreateFunc
	Repository           storage.TransactionalRepository
	TenantKey            string
	PollingInterval      time.Duration
	MaxParallelDeletions int
}

func (p *ServiceBindingUpdateInterceptorProvider) Provide() storage.UpdateAroundTxInterceptor {
	return &ServiceBindingInterceptor{
		osbClientCreateFunc:  p.OSBClientCreateFunc,
		repository:           p.Repository,
		tenantKey:            p.TenantKey,
		pollingInterval:      p.PollingInterval,
		maxParallelDeletions: p.MaxParallelDeletions,
	}
}

func (c *ServiceBindingUpdateInterceptorProvider) Name() string {
	return ServiceBindingUpdateInterceptorProviderName
}

const ServiceBindingDeleteInterceptorProviderName = "ServiceBindingDeleteInterceptorProvider"

// ServiceBindingDeleteInterceptorProvider provides an interceptor that notifies the actual broker about instance deletion
type ServiceBindingDeleteInterceptorProvider struct {
	OSBClientCreateFunc  osbc.CreateFunc
	Repository           storage.TransactionalRepository
	TenantKey            string
	PollingInterval      time.Duration
	MaxParallelDeletions int
}

func (p *ServiceBindingDeleteInterceptorProvider) Provide() storage.DeleteAroundTxInterceptor {
	return &ServiceBindingInterceptor{
		osbClientCreateFunc:  p.OSBClientCreateFunc,
		repository:           p.Repository,
		tenantKey:            p.TenantKey,
		pollingInterval:      p.PollingInterval,
		maxParallelDeletions: p.MaxParallelDeletions,
	}
}

func (c *ServiceBindingDeleteInterceptorProvider) Name() string {
	return ServiceBindingDeleteInterceptorProviderName
}

type ServiceBindingInterceptor struct {
	osbClientCreateFunc  osbc.CreateFunc
	repository           storage.TransactionalRepository
	tenantKey            string
	pollingInterval      time.Duration
	maxParallelDeletions int
}

func (*ServiceBindingInterceptor) AroundTxCreate(f storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return f
}

func (*ServiceBindingInterceptor) AroundTxUpdate(f storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return f
}

func (*ServiceBindingInterceptor) AroundTxDelete(f storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return f
}

//TODO orphan mitigation requirements are different !
