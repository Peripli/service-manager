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

package extension

import (
	"context"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

//
//type OperationType string
//
//const (
//	CreateOperation OperationType = "CREATE"
//	UpdateOperation OperationType = "UPDATE"
//	DeleteOpration  OperationType = "DELETE"
//)

type CreateHookFunc func(objectType types.ObjectType) CreateHook
type UpdateHookFunc func(objectType types.ObjectType) UpdateHook
type DeleteHookFunc func(objectType types.ObjectType) DeleteHook

type CreateHook interface {
	OnAPI(ctx context.Context, apiFunc func() (types.Object, error)) (types.Object, error)
	OnTransaction(ctx context.Context, txStorage storage.Warehouse, transactionFunc func() (types.Object, error)) error
	Supports(objectType types.ObjectType) bool
}

type UpdateHook interface {
	OnAPI(ctx context.Context, oldObject types.Object, apiFunc func(modifiedObject types.Object) (types.Object, error), changes ...*query.LabelChange) (types.Object, error)
	OnTransaction(ctx context.Context, storage storage.Warehouse, transactionFunc func() (oldObject, newObject types.Object, err error)) error
}

type DeleteHook interface {
	OnAPI(ctx context.Context, apiFunc func(ctx context.Context) error, criteria ...query.Criterion) error
	OnStorage(ctx context.Context, storage storage.Warehouse, transactionFunc func() (types.ObjectList, error)) error
}
