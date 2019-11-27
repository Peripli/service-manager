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
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

const ServiceInstanceCreateInterceptorName = "ServiceInstanceCreateInterceptor"

type ServiceInstanceCreateInsterceptorProvider struct {
	TenantIdentifier string
}

func (c *ServiceInstanceCreateInsterceptorProvider) Name() string {
	return ServiceInstanceCreateInterceptorName
}

func (c *ServiceInstanceCreateInsterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &serviceInstanceCreateInterceptor{
		TenantIdentifier: c.TenantIdentifier,
	}
}

type serviceInstanceCreateInterceptor struct {
	TenantIdentifier string
}

func (c *serviceInstanceCreateInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		serviceInstance := obj.(*types.ServiceInstance)

		tenantID := gjson.GetBytes([]byte(serviceInstance.Context), c.TenantIdentifier)
		if !tenantID.Exists() {
			log.D().Debugf("Could not add %s label to service instance with id %s. Label not found in OSB context.", c.TenantIdentifier, serviceInstance.ID)
			return h(ctx, serviceInstance)
		}

		labels := serviceInstance.GetLabels()
		if labels == nil {
			labels = types.Labels{}
		}
		labels[c.TenantIdentifier] = []string{tenantID.String()}

		serviceInstance.SetLabels(labels)

		return h(ctx, serviceInstance)
	}
}
