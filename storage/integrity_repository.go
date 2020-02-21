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

	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

// DataIntegrityDecorator decorates a repository to check the integrity of secured objects upon retrieval
func DataIntegrityDecorator(integrityProcessor security.IntegrityProcessor) TransactionalRepositoryDecorator {
	return func(next TransactionalRepository) (TransactionalRepository, error) {
		return NewIntegrityRepository(next, integrityProcessor), nil
	}
}

func NewIntegrityRepository(repository TransactionalRepository, integrityProcessor security.IntegrityProcessor) *TransactionalIntegrityRepository {
	return &TransactionalIntegrityRepository{
		integrityRepository: &integrityRepository{
			repository:         repository,
			integrityProcessor: integrityProcessor,
		},
		repository: repository,
	}
}

type integrityRepository struct {
	repository         Repository
	integrityProcessor security.IntegrityProcessor
}

type TransactionalIntegrityRepository struct {
	*integrityRepository
	repository TransactionalRepository
}

func (cr *integrityRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	if err := cr.setIntegrity(obj); err != nil {
		return nil, err
	}
	return cr.repository.Create(ctx, obj)
}

func (cr *integrityRepository) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	obj, err := cr.repository.Get(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}
	if err := cr.validateIntegrity(obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (cr *integrityRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objectList, err := cr.repository.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}
	for i := 0; i < objectList.Len(); i++ {
		item := objectList.ItemAt(i)
		if err := cr.validateIntegrity(item); err != nil {
			return nil, err
		}
	}
	return objectList, nil
}

func (cr *integrityRepository) Update(ctx context.Context, obj types.Object, labelChanges query.LabelChanges, criteria ...query.Criterion) (types.Object, error) {
	if err := cr.setIntegrity(obj); err != nil {
		return nil, err
	}
	return cr.repository.Update(ctx, obj, labelChanges, criteria...)
}

func (cr *integrityRepository) Count(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return cr.repository.Count(ctx, objectType, criteria...)
}

func (cr *integrityRepository) DeleteReturning(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	return cr.repository.DeleteReturning(ctx, objectType, criteria...)
}

func (cr *integrityRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) error {
	return cr.repository.Delete(ctx, objectType, criteria...)
}

// InTransaction wraps repository passed in the transaction to also validate integrity
func (cr *TransactionalIntegrityRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	return cr.repository.InTransaction(ctx, func(ctx context.Context, storage Repository) error {
		return f(ctx, &integrityRepository{
			repository:         storage,
			integrityProcessor: cr.integrityProcessor,
		})
	})
}

func (cr *integrityRepository) setIntegrity(obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		integrity, err := cr.integrityProcessor.CalculateIntegrity(securedObject)
		if err != nil {
			return err
		}
		securedObject.SetIntegrity(integrity)
	}
	return nil
}

func (cr *integrityRepository) validateIntegrity(obj types.Object) error {
	if securedObject, isSecured := obj.(types.Secured); isSecured {
		if !cr.integrityProcessor.ValidateIntegrity(securedObject) {
			return fmt.Errorf("invalid integrity for %s with ID %s", obj.GetType(), obj.GetID())
		}
	}
	return nil
}
