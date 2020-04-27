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
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

const ServiceInstanceCreateInterceptorName = "ServiceInstanceCreateInterceptor"

type createInterceptor struct {
	TenantIdentifier string
	EntityType       types.ObjectType

}

type ServiceInstanceCreateInsterceptorProvider struct {
	TenantIdentifier string
}

func (c *ServiceInstanceCreateInsterceptorProvider) Name() string {
	return ServiceInstanceCreateInterceptorName
}

func (c *ServiceInstanceCreateInsterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return &createInterceptor{
		TenantIdentifier: c.TenantIdentifier,
		EntityType: types.ServiceInstanceType,
	}
}


const ServiceBindingCreateInterceptorName = "ServiceBindingCreateInterceptor"

type ServiceBindingCreateInsterceptorProvider struct {
	TenantIdentifier string
}

func (c *ServiceBindingCreateInsterceptorProvider) Name() string {
	return ServiceBindingCreateInterceptorName
}

func (c *ServiceBindingCreateInsterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return &createInterceptor{
		TenantIdentifier: c.TenantIdentifier,
		EntityType: types.ServiceBindingType,
	}
}

func (c *createInterceptor) OnTxCreate(h storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {

		labels := obj.GetLabels()
		if labels == nil {
			labels = types.Labels{}
		}

		if _, ok := labels[c.TenantIdentifier]; ok {
			log.C(ctx).Debugf("Label %s is already set on service instance", c.TenantIdentifier)
			return h(ctx, storage, obj)
		}

		var context json.RawMessage
		if c.EntityType == types.ServiceInstanceType {
			context = obj.(*types.ServiceInstance).Context
		} else {
			context = obj.(*types.ServiceBinding).Context
		}

		tenantID := gjson.GetBytes(context, c.TenantIdentifier)
		if !tenantID.Exists() {
			log.C(ctx).Debugf("Could not add %s label to service instance with id %s. Label not found in OSB context.", c.TenantIdentifier, obj.GetID())
			return h(ctx, storage, obj)
		}
		labels[c.TenantIdentifier] = []string{tenantID.String()}
		obj.SetLabels(labels)

		return h(ctx, storage, obj)
	}
}
