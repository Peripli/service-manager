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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

// DataIntegrityDecorator decorates a repository to process the integrity of integral objects
func DataIntegrityDecorator(integrityProcessor security.IntegrityProcessor) TransactionalRepositoryDecorator {
	return func(next TransactionalRepository) (TransactionalRepository, error) {
		return NewIntegrityRepository(next, integrityProcessor), nil
	}
}

// NewIntegrityRepository returns a new TransactionIntegrityRepository using the specified integrity processor
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

// TransactionalIntegrityRepository is a TransactionalRepository which also processes the integrity of Integral objects
// before storing and after fetching them from the database
type TransactionalIntegrityRepository struct {
	*integrityRepository
	repository TransactionalRepository
}

func (cr *integrityRepository) QueryForList(ctx context.Context, objectType types.ObjectType, queryName NamedQuery, queryParams map[string]interface{}) (types.ObjectList, error) {
	objectList, err := cr.repository.QueryForList(ctx, objectType, queryName, queryParams)
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

func (cr *integrityRepository) GetForUpdate(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
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
	return cr.list(ctx, objectType, true, criteria...)
}

func (cr *integrityRepository) ListNoLabels(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	return cr.list(ctx, objectType, false, criteria...)
}

func (cr *integrityRepository) list(ctx context.Context, objectType types.ObjectType, withLabels bool, criteria ...query.Criterion) (types.ObjectList, error) {
	var (
		objectList types.ObjectList
		err        error
	)
	if withLabels {
		objectList, err = cr.repository.List(ctx, objectType, criteria...)
	} else {
		objectList, err = cr.repository.ListNoLabels(ctx, objectType, criteria...)
	}
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

func (cr *integrityRepository) Update(ctx context.Context, obj types.Object, labelChanges types.LabelChanges, criteria ...query.Criterion) (types.Object, error) {
	if err := cr.setIntegrity(obj); err != nil {
		return nil, err
	}
	return cr.repository.Update(ctx, obj, labelChanges, criteria...)
}

func (cr *integrityRepository) UpdateLabels(ctx context.Context, objectType types.ObjectType, objectID string, labelChanges types.LabelChanges, criteria ...query.Criterion) error {
	return cr.repository.UpdateLabels(ctx, objectType, objectID, labelChanges, criteria...)
}

func (cr *integrityRepository) Count(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return cr.repository.Count(ctx, objectType, criteria...)
}

func (cr *integrityRepository) CountLabelValues(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return cr.repository.CountLabelValues(ctx, objectType, criteria...)
}

func (cr *integrityRepository) DeleteReturning(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	return cr.repository.DeleteReturning(ctx, objectType, criteria...)
}

func (cr *integrityRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) error {
	return cr.repository.Delete(ctx, objectType, criteria...)
}

func (cr *TransactionalIntegrityRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	return cr.repository.InTransaction(ctx, func(ctx context.Context, storage Repository) error {
		return f(ctx, &integrityRepository{
			repository:         storage,
			integrityProcessor: cr.integrityProcessor,
		})
	})
}

func (cr *integrityRepository) GetEntities() []EntityMetadata {
	return cr.repository.GetEntities()
}

func (cr *integrityRepository) setIntegrity(obj types.Object) error {
	if integralObject, isIntegral := obj.(security.IntegralObject); isIntegral {
		integrity, err := cr.integrityProcessor.CalculateIntegrity(integralObject)
		if err != nil {
			return err
		}
		integralObject.SetIntegrity(integrity)
	}
	return nil
}

func (cr *integrityRepository) validateIntegrity(obj types.Object) error {
	if integralObject, isIntegral := obj.(security.IntegralObject); isIntegral {
		if !cr.integrityProcessor.ValidateIntegrity(integralObject) {
			return fmt.Errorf("invalid integrity for %s with ID %s", obj.GetType(), obj.GetID())
		}
	}
	return nil
}
