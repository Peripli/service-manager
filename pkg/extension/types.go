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
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type unionCreateHook struct {
	createHooks []CreateHook
}

func (c *unionCreateHook) AroundStorage(ctx context.Context, object types.Object) error {
	for _, h := range c.createHooks {
		if err := h.OnAPI(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func (c *unionCreateHook) InStorage(ctx context.Context, newObject types.Object, storage storage.Warehouse, transactionFunc func() error) error {
	for _, h := range c.createHooks {
		if err := h.OnTransaction(ctx, newObject, storage, transactionFunc); err != nil {
			return err
		}
	}
	return nil
}
func (c *unionCreateHook) Supports(objectType types.ObjectType) bool {
	for _, h := range c.createHooks {
		if !h.Supports(objectType) {
			return false
		}
	}
	return true
}

type unionUpdateHook struct {
	createHooks []CreateHook
}

func (c *unionUpdateHook) AroundStorage(ctx context.Context, object types.Object) error {
	for _, h := range c.createHooks {
		if err := h.OnAPI(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func (c *unionUpdateHook) InStorage(ctx context.Context, newObject types.Object, storage storage.Warehouse, transactionFunc func() error) error {
	for _, h := range c.createHooks {
		if err := h.OnTransaction(ctx, newObject, storage, transactionFunc); err != nil {
			return err
		}
	}
	return nil
}
func (c *unionUpdateHook) Supports(objectType types.ObjectType) bool {
	for _, h := range c.createHooks {
		if !h.Supports(objectType) {
			return false
		}
	}
	return true
}

func UnionCreateHook(objectType types.ObjectType, hook ...CreateHook) (CreateHookFunc, error) {
	unionHook := &unionCreateHook{createHooks: hook}
	if !unionHook.Supports(objectType) {
		return nil, fmt.Errorf("one of the create hooks does not support objects of type %s", objectType)
	}
	return func(objectType types.ObjectType) CreateHook {
		return unionHook
	}, nil
}

func UnionUpdateHook(objectType types.ObjectType, hook ...UpdateHook) (UpdateHookFunc, error) {
	unionHook := &unionUpdateHook{updateHooks: hook}
	if !unionHook.Supports(objectType) {
		return nil, fmt.Errorf("one of the update hooks does not support objects of type %s", objectType)
	}
	return func(objectType types.ObjectType) UpdateHook {
		return unionHook
	}, nil
}
