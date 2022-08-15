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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

const OperationSanitizerInterceptorName = "OperationSanitizerInterceptor"

type OperationSanitizerInterceptorProvider struct {
}

func (c *OperationSanitizerInterceptorProvider) Name() string {
	return OperationSanitizerInterceptorName
}

func (c *OperationSanitizerInterceptorProvider) Provide() storage.UpdateOnTxInterceptor {
	return &operationSanitizerInterceptor{}
}

type operationSanitizerInterceptor struct {
}

func (c *operationSanitizerInterceptor) OnTxUpdate(h storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
		op := newObj.(*types.Operation)
		if op.State == types.SUCCEEDED || op.State == types.FAILED {
			if op.Context != nil && op.Context.UserInfo != nil {
				op.Context.UserInfo = nil
			}
		}
		return h(ctx, txStorage, oldObj, newObj, labelChanges...)
	}
}
