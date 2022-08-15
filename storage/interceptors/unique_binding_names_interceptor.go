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
	UniqueBindingNameCreateInterceptorName = "UniqueBindingNameCreateInterceptor"
)

// UniqueBindingNameCreateInterceptorProvider provides an interceptor that forbids creation of bindings with the same name in a given tenant
type UniqueBindingNameCreateInterceptorProvider struct {
	Repository storage.TransactionalRepository
}

func (c *UniqueBindingNameCreateInterceptorProvider) Name() string {
	return UniqueBindingNameCreateInterceptorName
}

func (c *UniqueBindingNameCreateInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &uniqueBindingNameInterceptor{
		Repository: c.Repository,
	}
}

type uniqueBindingNameInterceptor struct {
	Repository storage.TransactionalRepository
}

func (c *uniqueBindingNameInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		if err := c.checkUniqueName(ctx, obj.(*types.ServiceBinding)); err != nil {
			return nil, err
		}
		return h(ctx, obj)
	}
}

func (c *uniqueBindingNameInterceptor) checkUniqueName(ctx context.Context, binding *types.ServiceBinding) error {
	operation, operationFound := opcontext.Get(ctx)
	if !operationFound {
		log.C(ctx).Debug("operation missing from context")
	}

	if operationFound && operation.Reschedule {
		log.C(ctx).Info("skipping unique check of binding name as this is a rescheduled operation")
		return nil
	}

	countCriteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "service_instance_id", binding.ServiceInstanceID),
		query.ByField(query.EqualsOperator, nameProperty, binding.Name),
	}
	bindingCount, err := c.Repository.Count(ctx, types.ServiceBindingType, countCriteria...)
	if err != nil {
		return fmt.Errorf("could not get count of service bindings %s", err)
	}

	if bindingCount > 0 {
		return &util.HTTPError{
			ErrorType:   "Conflict",
			Description: fmt.Sprintf("binding with same name exists for instance with id %s", binding.ServiceInstanceID),
			StatusCode:  http.StatusConflict,
		}
	}

	return nil
}
