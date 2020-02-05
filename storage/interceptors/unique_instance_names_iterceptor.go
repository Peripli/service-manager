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
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

const (
	UniqueInstanceNameCreateInterceptorName = "UniqueInstanceNameCreateInterceptor"
	UniqueInstanceNameUpdateInterceptorName = "UniqueInstanceNameUpdateInterceptor"
	nameProperty                            = "name"
)

// UniqueInstanceNameCreateInterceptorProvider provides an interceptor that forbids creation of instances with the same name in a given tenant
type UniqueInstanceNameCreateInterceptorProvider struct {
	TenantIdentifier string
	Repository       storage.TransactionalRepository
}

func (c *UniqueInstanceNameCreateInterceptorProvider) Name() string {
	return UniqueInstanceNameCreateInterceptorName
}

func (c *UniqueInstanceNameCreateInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &uniqueInstanceNameInterceptor{
		TenantIdentifier: c.TenantIdentifier,
		Repository:       c.Repository,
	}
}

// UniqueInstanceNameUpdateInterceptorProvider provides an interceptor that forbids updating an instance name that breaks uniqueness in a given tenant
type UniqueInstanceNameUpdateInterceptorProvider struct {
	TenantIdentifier string
	Repository       storage.TransactionalRepository
}

func (c *UniqueInstanceNameUpdateInterceptorProvider) Name() string {
	return UniqueInstanceNameUpdateInterceptorName
}

func (c *UniqueInstanceNameUpdateInterceptorProvider) Provide() storage.UpdateAroundTxInterceptor {
	return &uniqueInstanceNameInterceptor{
		TenantIdentifier: c.TenantIdentifier,
		Repository:       c.Repository,
	}
}

type uniqueInstanceNameInterceptor struct {
	TenantIdentifier string
	Repository       storage.TransactionalRepository
}

func (c *uniqueInstanceNameInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (object types.Object, err error) {
		if err := c.checkUniqueName(ctx, obj.GetLabels(), obj.(*types.ServiceInstance)); err != nil {
			return nil, err
		}
		return h(ctx, obj)
	}
}

func (c *uniqueInstanceNameInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, newObj types.Object, labelChanges ...*query.LabelChange) (object types.Object, err error) {
		oldObj, err := c.Repository.Get(ctx, types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", newObj.GetID()))
		if err != nil {
			return nil, err
		}
		oldInstance := oldObj.(*types.ServiceInstance)
		newInstance := newObj.(*types.ServiceInstance)
		if newInstance.Name != oldInstance.Name {
			if err := c.checkUniqueName(ctx, oldObj.GetLabels(), newInstance); err != nil {
				return nil, err
			}
		}
		return h(ctx, newObj, labelChanges...)
	}
}

func (c *uniqueInstanceNameInterceptor) checkUniqueName(ctx context.Context, labels types.Labels, instance *types.ServiceInstance) error {
	if instance.PlatformID != types.SMPlatform {
		return nil
	}
	countCriteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, nameProperty, instance.Name),
	}
	if len(c.TenantIdentifier) != 0 {
		if labels == nil {
			labels = types.Labels{}
		}
		tenantIDLabelValue, ok := labels[c.TenantIdentifier]
		if ok && len(tenantIDLabelValue) != 0 {
			tenantID := tenantIDLabelValue[0]
			countCriteria = append(countCriteria, query.ByLabel(query.EqualsOperator, c.TenantIdentifier, tenantID))
		}
	}
	instanceCount, err := c.Repository.Count(ctx, types.ServiceInstanceType, countCriteria...)

	if err != nil {
		return fmt.Errorf("could not get count of service instances %s", err)
	}
	if instanceCount > 0 {
		return &util.HTTPError{
			ErrorType:   "Conflict",
			Description: "instance with same name exists for the current tenant",
			StatusCode:  http.StatusConflict,
		}
	}
	return nil
}
