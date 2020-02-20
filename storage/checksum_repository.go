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

package storage

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type checksumRepository struct {
	TransactionalRepository
	checksumFunc func(data []byte) [32]byte
}

func NewChecksumDecorator(checksumFunc func(data []byte) [32]byte) TransactionalRepositoryDecorator {
	return func(next TransactionalRepository) (TransactionalRepository, error) {
		return &checksumRepository{
			TransactionalRepository: next,
			checksumFunc:            checksumFunc,
		}, nil
	}
}

func (cr *checksumRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	cr.setCheckSum(obj)
	return cr.TransactionalRepository.Create(ctx, obj)
}

func (cr *checksumRepository) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	obj, err := cr.TransactionalRepository.Get(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	if err := cr.validateChecksum(obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (cr *checksumRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objectList, err := cr.TransactionalRepository.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}
	// TODO: with this decorator pattern we need to iterate again over all fetched entities...
	for i := 0; i < objectList.Len(); i++ {
		item := objectList.ItemAt(i)
		if err := cr.validateChecksum(item); err != nil {
			return nil, err
		}
	}
	return objectList, nil
}

func (cr *checksumRepository) Update(ctx context.Context, obj types.Object, labelChanges query.LabelChanges, criteria ...query.Criterion) (types.Object, error) {
	cr.setCheckSum(obj)
	return cr.TransactionalRepository.Update(ctx, obj, labelChanges, criteria...)
}

func (cr *checksumRepository) setCheckSum(obj types.Object) {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		securedObject.SetChecksum(cr.checksumFunc)
	}
}

func (cr *checksumRepository) validateChecksum(obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		if !securedObject.ValidateChecksum(cr.checksumFunc) {
			return fmt.Errorf("invalid checksum for %s with ID %s", obj.GetType(), obj.GetID())
		}
	}
	return nil
}
