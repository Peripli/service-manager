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

package opcontext

import (
	"context"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

// operationCtxKey allows putting the currently running operation is the context. This is required for
// some interceptors - based on the operation they execute different logic or they might update the actual operation
type operationCtxKey struct{}

func Get(ctx context.Context) (*types.Operation, bool) {
	currentOperation := ctx.Value(operationCtxKey{})
	if currentOperation == nil {
		return nil, false
	}
	return currentOperation.(*types.Operation), true
}

func Set(ctx context.Context, operation *types.Operation) (context.Context, error) {
	if err := operation.Validate(); err != nil {
		return nil, err
	}

	return context.WithValue(ctx, operationCtxKey{}, operation), nil
}
