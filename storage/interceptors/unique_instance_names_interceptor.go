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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/operations/opcontext"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
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
		if err := c.checkUniqueName(ctx, obj.(*types.ServiceInstance)); err != nil {
			return nil, err
		}
		return h(ctx, obj)
	}
}

func (c *uniqueInstanceNameInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, newObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, err error) {
		oldObj, err := c.Repository.Get(ctx, types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", newObj.GetID()))
		if err != nil {
			return nil, err
		}
		oldInstance := oldObj.(*types.ServiceInstance)
		newInstance := newObj.(*types.ServiceInstance)
		if newInstance.Name != oldInstance.Name {
			if err := c.checkUniqueName(ctx, newInstance); err != nil {
				return nil, err
			}
		}
		return h(ctx, newObj, labelChanges...)
	}
}

func (c *uniqueInstanceNameInterceptor) checkUniqueName(ctx context.Context, instance *types.ServiceInstance) error {
	operation, operationFound := opcontext.Get(ctx)
	if !operationFound {
		log.C(ctx).Debug("operation missing from context")
	}

	rescheduledOperation := operationFound && operation.Reschedule
	if instance.PlatformID != types.SMPlatform || rescheduledOperation {
		if rescheduledOperation {
			log.C(ctx).Info("skipping unique check of instance name as this is a rescheduled operation")
		}
		return nil
	}
	countCriteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, nameProperty, instance.Name),
	}

	criteriaForContext := query.CriteriaForContext(ctx)
	for _, criterion := range criteriaForContext {
		// use labelQuery criteria from context if exist as they provide the scope for name uniqueness, e.g. tenant scope
		if criterion.Type == query.LabelQuery {
			countCriteria = append(countCriteria, criterion)
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
