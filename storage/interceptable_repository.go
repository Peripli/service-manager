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
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

func NewInterceptableTransactionalRepository(repository TransactionalRepository) *InterceptableTransactionalRepository {
	return &InterceptableTransactionalRepository{
		smStorageRepository: repository,
		createProviders:     make(map[types.ObjectType][]OrderedCreateInterceptorProvider),
		updateProviders:     make(map[types.ObjectType][]OrderedUpdateInterceptorProvider),
		deleteProviders:     make(map[types.ObjectType][]OrderedDeleteInterceptorProvider),
	}
}

func newInterceptableRepository(repository Repository,
	providedCreateInterceptors map[types.ObjectType]CreateInterceptor,
	providedUpdateInterceptors map[types.ObjectType]UpdateInterceptor,
	providedDeleteInterceptors map[types.ObjectType]DeleteInterceptor) *interceptableRepository {

	return &interceptableRepository{
		repositoryInTransaction: repository,
		createInterceptor:       providedCreateInterceptors,
		updateInterceptor:       providedUpdateInterceptors,
		deleteInterceptor:       providedDeleteInterceptors,
	}
}

type InterceptableTransactionalRepository struct {
	smStorageRepository TransactionalRepository

	createProviders map[types.ObjectType][]OrderedCreateInterceptorProvider
	updateProviders map[types.ObjectType][]OrderedUpdateInterceptorProvider
	deleteProviders map[types.ObjectType][]OrderedDeleteInterceptorProvider
}

type interceptableRepository struct {
	repositoryInTransaction Repository

	createInterceptor map[types.ObjectType]CreateInterceptor
	updateInterceptor map[types.ObjectType]UpdateInterceptor
	deleteInterceptor map[types.ObjectType]DeleteInterceptor
}

func (ir *interceptableRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	objectType := obj.GetType()

	createObjectFunc := func(ctx context.Context, _ Repository, newObject types.Object) (types.Object, error) {

		createdObj, err := ir.repositoryInTransaction.Create(ctx, newObject)
		if err != nil {
			return nil, err
		}

		return createdObj, nil
	}

	var createdObj types.Object
	var err error
	if createInterceptorChain, found := ir.createInterceptor[objectType]; found {
		// remove the create interceptor chain so that if one of the interceptors in the chain tries
		// to create another resource of the same type we don't get into infinite recursion

		// clean up to avoid nested infinite chain
		delete(ir.createInterceptor, objectType)

		createdObj, err = createInterceptorChain.OnTxCreate(createObjectFunc)(ctx, ir, obj)

		// restore the chain
		ir.createInterceptor[objectType] = createInterceptorChain
	} else {
		createdObj, err = createObjectFunc(ctx, ir.repositoryInTransaction, obj)
	}

	if err != nil {
		return nil, err
	}

	return createdObj, nil
}

func (ir *interceptableRepository) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	object, err := ir.repositoryInTransaction.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (ir *interceptableRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objectList, err := ir.repositoryInTransaction.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	return objectList, nil
}

func (ir *interceptableRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	deleteObjectFunc := func(ctx context.Context, _ Repository, _ types.ObjectList, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		objectList, err := ir.repositoryInTransaction.Delete(ctx, objectType, deletionCriteria...)
		if err != nil {
			return nil, err
		}

		return objectList, nil
	}

	var objectList types.ObjectList
	var objects types.ObjectList
	var err error

	if deleteInterceptorChain, found := ir.deleteInterceptor[objectType]; found {
		objects, err = ir.List(ctx, objectType, criteria...)
		if err != nil {
			return nil, err
		}

		delete(ir.deleteInterceptor, objectType)

		objectList, err = deleteInterceptorChain.OnTxDelete(deleteObjectFunc)(ctx, ir, objects, criteria...)

		ir.deleteInterceptor[objectType] = deleteInterceptorChain

	} else {
		objectList, err = deleteObjectFunc(ctx, nil, nil, criteria...)
	}

	if err != nil {
		return nil, err
	}

	return objectList, nil
}

func (ir *interceptableRepository) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	objectType := obj.GetType()

	updateObjFunc := func(ctx context.Context, _ Repository, oldObj, newObj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		newObj.SetUpdatedAt(time.Now().UTC())

		object, err := ir.repositoryInTransaction.Update(ctx, newObj, labelChanges...)
		if err != nil {
			return nil, err
		}

		labels, _, _ := query.ApplyLabelChangesToLabels(labelChanges, newObj.GetLabels())
		object.SetLabels(labels)

		return object, nil
	}

	var updatedObj types.Object
	var err error

	// postgres storage implementation also locks the retrieved row for update
	oldObj, err := ir.Get(ctx, objectType, obj.GetID())
	if err != nil {
		return nil, err
	}

	// while the AroundTx hooks were being executed the stored resource actually changed - another concurrent update
	// happened and finished concurrently and before this one so fail the request
	if util.ToRFCFormat(oldObj.GetUpdatedAt()) != util.ToRFCFormat(obj.GetUpdatedAt()) {
		return nil, util.ErrConcurrentResourceModification
	}

	if updateInterceptorChain, found := ir.updateInterceptor[objectType]; found {
		delete(ir.updateInterceptor, objectType)

		updatedObj, err = updateInterceptorChain.OnTxUpdate(updateObjFunc)(ctx, ir, oldObj, obj, labelChanges...)

		ir.updateInterceptor[objectType] = updateInterceptorChain

	} else {
		updatedObj, err = updateObjFunc(ctx, ir, oldObj, obj, labelChanges...)
	}

	if err != nil {
		return nil, err
	}

	return updatedObj, nil
}

func (itr *InterceptableTransactionalRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	createInterceptors, updateInterceptors, deleteInterceptors := itr.provideInterceptors()

	fWrapper := func(ctx context.Context, storage Repository) error {
		wrappedStorage := newInterceptableRepository(storage, createInterceptors, updateInterceptors, deleteInterceptors)
		return f(ctx, wrappedStorage)
	}

	return itr.smStorageRepository.InTransaction(ctx, fWrapper)
}

func (itr *InterceptableTransactionalRepository) AddCreateInterceptorProvider(objectType types.ObjectType, provider OrderedCreateInterceptorProvider) {
	itr.validateCreateProviders(objectType, provider.Name(), provider.InterceptorOrder)
	itr.createProviders[objectType] = append(itr.createProviders[objectType], provider)
}

func (itr *InterceptableTransactionalRepository) AddUpdateInterceptorProvider(objectType types.ObjectType, provider OrderedUpdateInterceptorProvider) {
	itr.validateUpdateProviders(objectType, provider.Name(), provider.InterceptorOrder)
	itr.updateProviders[objectType] = append(itr.updateProviders[objectType], provider)
}

func (itr *InterceptableTransactionalRepository) AddDeleteInterceptorProvider(objectType types.ObjectType, provider OrderedDeleteInterceptorProvider) {
	itr.validateDeleteProviders(objectType, provider.Name(), provider.InterceptorOrder)
	itr.deleteProviders[objectType] = append(itr.deleteProviders[objectType], provider)
}

type finalCreateObjectInterceptor struct {
	repository                 TransactionalRepository
	objectType                 types.ObjectType
	providedCreateInterceptors map[types.ObjectType]CreateInterceptor
	providedUpdateInterceptors map[types.ObjectType]UpdateInterceptor
	providedDeleteInterceptors map[types.ObjectType]DeleteInterceptor
}

func (final *finalCreateObjectInterceptor) InterceptCreateOnTx(ctx context.Context, txStorage Repository, newObject types.Object) (types.Object, error) {
	createdObj, err := txStorage.Create(ctx, newObject)
	if err != nil {
		return nil, err
	}

	return createdObj, nil
}

func (final *finalCreateObjectInterceptor) InterceptCreateAroundTx(ctx context.Context, obj types.Object) (types.Object, error) {
	var createdObj types.Object
	var err error

	if err := final.repository.InTransaction(ctx, func(ctx context.Context, txStorage Repository) error {
		interceptableRepository := newInterceptableRepository(txStorage, final.providedCreateInterceptors, final.providedUpdateInterceptors, final.providedDeleteInterceptors)
		createdObj, err = interceptableRepository.Create(ctx, obj)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return createdObj, nil
}

func (itr *InterceptableTransactionalRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors := itr.provideInterceptors()

	objectType := obj.GetType()

	finalInterceptor := &finalCreateObjectInterceptor{
		repository:                 itr.smStorageRepository,
		objectType:                 objectType,
		providedCreateInterceptors: providedCreateInterceptors,
		providedUpdateInterceptors: providedUpdateInterceptors,
		providedDeleteInterceptors: providedDeleteInterceptors,
	}

	var err error
	if providedCreateInterceptors[objectType] != nil {
		obj, err = providedCreateInterceptors[objectType].AroundTxCreate(finalInterceptor.InterceptCreateAroundTx)(ctx, obj)
	} else {
		obj, err = finalInterceptor.InterceptCreateAroundTx(ctx, obj)
	}

	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (itr *InterceptableTransactionalRepository) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	object, err := itr.smStorageRepository.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (itr *InterceptableTransactionalRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objectList, err := itr.smStorageRepository.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	return objectList, nil
}

type finalDeleteObjectInterceptor struct {
	repository                 TransactionalRepository
	objectType                 types.ObjectType
	providedCreateInterceptors map[types.ObjectType]CreateInterceptor
	providedUpdateInterceptors map[types.ObjectType]UpdateInterceptor
	providedDeleteInterceptors map[types.ObjectType]DeleteInterceptor
}

func (final *finalDeleteObjectInterceptor) InterceptDeleteOnTx(ctx context.Context, txStorage Repository, criteria ...query.Criterion) (types.ObjectList, error) {
	return txStorage.Delete(ctx, final.objectType, criteria...)
}

func (final *finalDeleteObjectInterceptor) InterceptDeleteAroundTx(ctx context.Context, criteria ...query.Criterion) (types.ObjectList, error) {
	var result types.ObjectList
	var err error

	if err := final.repository.InTransaction(ctx, func(ctx context.Context, txStorage Repository) error {
		interceptableRepository := newInterceptableRepository(txStorage, final.providedCreateInterceptors, final.providedUpdateInterceptors, final.providedDeleteInterceptors)
		result, err = interceptableRepository.Delete(ctx, final.objectType, criteria...)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (itr *InterceptableTransactionalRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors := itr.provideInterceptors()

	finalInterceptor := &finalDeleteObjectInterceptor{
		repository:                 itr.smStorageRepository,
		objectType:                 objectType,
		providedCreateInterceptors: providedCreateInterceptors,
		providedUpdateInterceptors: providedUpdateInterceptors,
		providedDeleteInterceptors: providedDeleteInterceptors,
	}

	var objectList types.ObjectList
	var err error

	if providedDeleteInterceptors[objectType] != nil {
		objectList, err = providedDeleteInterceptors[objectType].AroundTxDelete(finalInterceptor.InterceptDeleteAroundTx)(ctx, criteria...)
	} else {
		objectList, err = finalInterceptor.InterceptDeleteAroundTx(ctx, criteria...)
	}

	if err != nil {
		return nil, err
	}

	return objectList, err
}

type finalUpdateObjectInterceptor struct {
	repository                 TransactionalRepository
	objectType                 types.ObjectType
	providedCreateInterceptors map[types.ObjectType]CreateInterceptor
	providedUpdateInterceptors map[types.ObjectType]UpdateInterceptor
	providedDeleteInterceptors map[types.ObjectType]DeleteInterceptor
}

func (final *finalUpdateObjectInterceptor) InterceptUpdateOnTxFunc(ctx context.Context, txStorage Repository, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	return txStorage.Update(ctx, obj, labelChanges...)
}

func (final *finalUpdateObjectInterceptor) InterceptUpdateAroundTxFunc(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	var err error
	var result types.Object

	if err = final.repository.InTransaction(ctx, func(ctx context.Context, txStorage Repository) error {
		interceptableRepository := newInterceptableRepository(txStorage, final.providedCreateInterceptors, final.providedUpdateInterceptors, final.providedDeleteInterceptors)

		result, err = interceptableRepository.Update(ctx, obj, labelChanges...)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (itr *InterceptableTransactionalRepository) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors := itr.provideInterceptors()

	objectType := obj.GetType()

	finalInterceptor := &finalUpdateObjectInterceptor{
		repository:                 itr.smStorageRepository,
		objectType:                 objectType,
		providedCreateInterceptors: providedCreateInterceptors,
		providedUpdateInterceptors: providedUpdateInterceptors,
		providedDeleteInterceptors: providedDeleteInterceptors,
	}

	var err error
	if providedUpdateInterceptors[objectType] != nil {
		obj, err = providedUpdateInterceptors[objectType].AroundTxUpdate(finalInterceptor.InterceptUpdateAroundTxFunc)(ctx, obj, labelChanges...)
	} else {
		obj, err = finalInterceptor.InterceptUpdateAroundTxFunc(ctx, obj, labelChanges...)
	}

	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (itr *InterceptableTransactionalRepository) validateCreateProviders(objectType types.ObjectType, providerName string, order InterceptorOrder) {
	var existingProviderNames []string
	for _, existingProvider := range itr.createProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}

	validateProviderOrder(order, existingProviderNames, providerName)
}

func (itr *InterceptableTransactionalRepository) validateUpdateProviders(objectType types.ObjectType, providerName string, order InterceptorOrder) {
	var existingProviderNames []string
	for _, existingProvider := range itr.updateProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}

	validateProviderOrder(order, existingProviderNames, providerName)
}

func (itr *InterceptableTransactionalRepository) validateDeleteProviders(objectType types.ObjectType, providerName string, order InterceptorOrder) {
	var existingProviderNames []string
	for _, existingProvider := range itr.deleteProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}

	validateProviderOrder(order, existingProviderNames, providerName)
}

func validateProviderOrder(order InterceptorOrder, existingProviderNames []string, providerName string) {
	positionAroundTx, aroundTxName := order.AroundTxPosition.PositionType, order.AroundTxPosition.Name
	positionTx, nameTx := order.OnTxPosition.PositionType, order.OnTxPosition.Name
	if providerWithNameExists(existingProviderNames, providerName) {
		log.D().Panicf("%s create interceptor provider is already registered", providerName)
	}

	if positionAroundTx != PositionNone {
		if !providerWithNameExists(existingProviderNames, aroundTxName) {
			log.D().Panicf("could not find interceptor with name %s", aroundTxName)
		}
	}
	if positionTx != PositionNone {
		if !providerWithNameExists(existingProviderNames, nameTx) {
			log.D().Panicf("could not find interceptor with name %s", nameTx)
		}
	}
}

func providerWithNameExists(existingNames []string, orderedRelativeTo string) bool {
	for _, name := range existingNames {
		if name == orderedRelativeTo {
			return true
		}
	}
	return false
}

func (itr *InterceptableTransactionalRepository) provideInterceptors() (map[types.ObjectType]CreateInterceptor, map[types.ObjectType]UpdateInterceptor, map[types.ObjectType]DeleteInterceptor) {
	providedCreateInterceptors := make(map[types.ObjectType]CreateInterceptor)
	for objectType, providers := range itr.createProviders {
		providedCreateInterceptors[objectType] = newCreateInterceptorChain(providers)
	}
	providedUpdateInterceptors := make(map[types.ObjectType]UpdateInterceptor)
	for objectType, providers := range itr.updateProviders {
		providedUpdateInterceptors[objectType] = newUpdateInterceptorChain(providers)

	}
	providedDeleteInterceptors := make(map[types.ObjectType]DeleteInterceptor)
	for objectType, providers := range itr.deleteProviders {
		providedDeleteInterceptors[objectType] = newDeleteInterceptorChain(providers)
	}
	return providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors
}
