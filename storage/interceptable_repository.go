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
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

const updateInProgress string = "update_in_progress"

func NewInterceptableTransactionalRepository(repository TransactionalRepository) *InterceptableTransactionalRepository {
	return &InterceptableTransactionalRepository{
		smStorageRepository: repository,

		orderedCreateAroundTxProvidersNames: make(map[types.ObjectType][]string),
		orderedCreateOnTxProvidersNames:     make(map[types.ObjectType][]string),
		createProviders:                     make(map[types.ObjectType]map[string]OrderedCreateInterceptorProvider),
		createAroundTxProviders:             make(map[types.ObjectType]map[string]OrderedCreateAroundTxInterceptorProvider),
		createOnTxProviders:                 make(map[types.ObjectType]map[string]OrderedCreateOnTxInterceptorProvider),

		orderedUpdateAroundTxProvidersNames: make(map[types.ObjectType][]string),
		orderedUpdateOnTxProvidersNames:     make(map[types.ObjectType][]string),
		updateProviders:                     make(map[types.ObjectType]map[string]OrderedUpdateInterceptorProvider),
		updateAroundTxProviders:             make(map[types.ObjectType]map[string]OrderedUpdateAroundTxInterceptorProvider),
		updateOnTxProviders:                 make(map[types.ObjectType]map[string]OrderedUpdateOnTxInterceptorProvider),

		orderedDeleteAroundTxProvidersNames: make(map[types.ObjectType][]string),
		orderedDeleteOnTxProvidersNames:     make(map[types.ObjectType][]string),
		deleteProviders:                     make(map[types.ObjectType]map[string]OrderedDeleteInterceptorProvider),
		deleteAroundTxProviders:             make(map[types.ObjectType]map[string]OrderedDeleteAroundTxInterceptorProvider),
		deleteOnTxProviders:                 make(map[types.ObjectType]map[string]OrderedDeleteOnTxInterceptorProvider),
	}
}

func newScopedRepositoryWithOnTxInterceptors(repository Repository,
	providedCreateInterceptors map[types.ObjectType]func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc,
	providedUpdateInterceptors map[types.ObjectType]func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc,
	providedDeleteInterceptors map[types.ObjectType]func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc) *queryScopedInterceptableRepository {

	return &queryScopedInterceptableRepository{
		repositoryInTransaction: repository,
		createOnTxFuncs:         providedCreateInterceptors,
		updateOnTxFuncs:         providedUpdateInterceptors,
		deleteOnTxFuncs:         providedDeleteInterceptors,
	}
}

func newScopedRepositoryWithInterceptors(repository Repository,
	providedCreateInterceptors map[types.ObjectType]CreateInterceptor,
	providedUpdateInterceptors map[types.ObjectType]UpdateInterceptor,
	providedDeleteInterceptors map[types.ObjectType]DeleteInterceptor) *queryScopedInterceptableRepository {
	createOnTxFuncs := make(map[types.ObjectType]func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc, len(providedCreateInterceptors))
	for objType, interceptor := range providedCreateInterceptors {
		createOnTxFuncs[objType] = interceptor.OnTxCreate
	}
	updateOnTxFuncs := make(map[types.ObjectType]func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc, len(providedCreateInterceptors))
	for objType, interceptor := range providedUpdateInterceptors {
		updateOnTxFuncs[objType] = interceptor.OnTxUpdate
	}
	deleteOnTxFuncs := make(map[types.ObjectType]func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc, len(providedCreateInterceptors))
	for objType, interceptor := range providedDeleteInterceptors {
		deleteOnTxFuncs[objType] = interceptor.OnTxDelete
	}
	return newScopedRepositoryWithOnTxInterceptors(repository, createOnTxFuncs, updateOnTxFuncs, deleteOnTxFuncs)
}

type InterceptableTransactionalRepository struct {
	smStorageRepository TransactionalRepository

	orderedCreateAroundTxProvidersNames map[types.ObjectType][]string
	orderedCreateOnTxProvidersNames     map[types.ObjectType][]string
	createProviders                     map[types.ObjectType]map[string]OrderedCreateInterceptorProvider
	createAroundTxProviders             map[types.ObjectType]map[string]OrderedCreateAroundTxInterceptorProvider
	createOnTxProviders                 map[types.ObjectType]map[string]OrderedCreateOnTxInterceptorProvider

	orderedUpdateAroundTxProvidersNames map[types.ObjectType][]string
	orderedUpdateOnTxProvidersNames     map[types.ObjectType][]string
	updateProviders                     map[types.ObjectType]map[string]OrderedUpdateInterceptorProvider
	updateAroundTxProviders             map[types.ObjectType]map[string]OrderedUpdateAroundTxInterceptorProvider
	updateOnTxProviders                 map[types.ObjectType]map[string]OrderedUpdateOnTxInterceptorProvider

	orderedDeleteAroundTxProvidersNames map[types.ObjectType][]string
	orderedDeleteOnTxProvidersNames     map[types.ObjectType][]string
	deleteProviders                     map[types.ObjectType]map[string]OrderedDeleteInterceptorProvider
	deleteAroundTxProviders             map[types.ObjectType]map[string]OrderedDeleteAroundTxInterceptorProvider
	deleteOnTxProviders                 map[types.ObjectType]map[string]OrderedDeleteOnTxInterceptorProvider
}

// queryScopedInterceptableRepository wraps a Repository to be used throughout a transaction (in all OnTx interceptors).
// It also holds sets of interceptors for each object type to run inside the transaction lifecycle. The repository is
// query scoped meaning that a new instance must be created in each repository operation
type queryScopedInterceptableRepository struct {
	repositoryInTransaction Repository

	createOnTxFuncs map[types.ObjectType]func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc
	updateOnTxFuncs map[types.ObjectType]func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc
	deleteOnTxFuncs map[types.ObjectType]func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc
}

func (ir *queryScopedInterceptableRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	createObjectFunc := func(ctx context.Context, _ Repository, newObject types.Object) (types.Object, error) {
		createdObj, err := ir.repositoryInTransaction.Create(ctx, newObject)
		if err != nil {
			return nil, err
		}

		return createdObj, nil
	}

	var createdObj types.Object
	var err error
	objectType := obj.GetType()
	if createOnTxFunc, found := ir.createOnTxFuncs[objectType]; found {
		// remove the create interceptor chain so that if one of the interceptors in the chain tries
		// to create another resource of the same type we don't get into infinite recursion

		// clean up to avoid nested infinite chain
		delete(ir.createOnTxFuncs, objectType)

		createdObj, err = createOnTxFunc(createObjectFunc)(ctx, ir, obj)

		// restore the chain
		ir.createOnTxFuncs[objectType] = createOnTxFunc
	} else {
		createdObj, err = createObjectFunc(ctx, ir.repositoryInTransaction, obj)
	}

	if err != nil {
		return nil, err
	}

	return createdObj, nil
}

func (ir *queryScopedInterceptableRepository) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	object, err := ir.repositoryInTransaction.Get(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (ir *queryScopedInterceptableRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objectList, err := ir.repositoryInTransaction.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}

	return objectList, nil
}

func (ir *queryScopedInterceptableRepository) Count(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return ir.repositoryInTransaction.Count(ctx, objectType, criteria...)
}

func (ir *queryScopedInterceptableRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
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

	if deleteOnTxFunc, found := ir.deleteOnTxFuncs[objectType]; found {
		objects, err = ir.List(ctx, objectType, criteria...)
		if err != nil {
			return nil, err
		}

		delete(ir.deleteOnTxFuncs, objectType)

		objectList, err = deleteOnTxFunc(deleteObjectFunc)(ctx, ir, objects, criteria...)

		ir.deleteOnTxFuncs[objectType] = deleteOnTxFunc

	} else {
		objectList, err = deleteObjectFunc(ctx, nil, nil, criteria...)
	}

	if err != nil {
		return nil, err
	}

	return objectList, nil
}

func (ir *queryScopedInterceptableRepository) Update(ctx context.Context, obj types.Object, labelChanges query.LabelChanges, criteria ...query.Criterion) (types.Object, error) {
	updateObjFunc := func(ctx context.Context, _ Repository, oldObj, newObj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		newObj.SetUpdatedAt(time.Now().UTC())

		object, err := ir.repositoryInTransaction.Update(ctx, newObj, labelChanges, criteria...)
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
	objectType := obj.GetType()
	byID := query.ByField(query.EqualsOperator, "id", obj.GetID())
	oldObj, err := ir.Get(ctx, objectType, byID)
	if err != nil {
		return nil, err
	}

	// while the AroundTx hooks were being executed the stored resource actually changed - another concurrent update
	// happened and finished concurrently and before this one so fail the request
	// update to the same entity in the same transaction may be possible from an interceptor
	inUpdate, _ := ctx.Value(updateInProgress).(bool)
	if !oldObj.GetUpdatedAt().UTC().Equal(obj.GetUpdatedAt().UTC()) && !inUpdate {
		return nil, util.ErrConcurrentResourceModification
	}

	if updateOnTxFunc, found := ir.updateOnTxFuncs[objectType]; found {
		delete(ir.updateOnTxFuncs, objectType)

		// specify that an update is in progress so that multiple updates to the entity are not taken as false positive concurrent update
		ctx = context.WithValue(ctx, updateInProgress, true)
		updatedObj, err = updateOnTxFunc(updateObjFunc)(ctx, ir, oldObj, obj, labelChanges...)

		ir.updateOnTxFuncs[objectType] = updateOnTxFunc

	} else {
		updatedObj, err = updateObjFunc(ctx, ir, oldObj, obj, labelChanges...)
	}

	if err != nil {
		return nil, err
	}

	return updatedObj, nil
}

func (itr *InterceptableTransactionalRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	createOnTxInterceptors, updateOnTxInterceptors, deleteOnTxInterceptors := itr.provideOnTxInterceptors()

	fWrapper := func(ctx context.Context, storage Repository) error {
		wrappedStorage := newScopedRepositoryWithOnTxInterceptors(storage, createOnTxInterceptors, updateOnTxInterceptors, deleteOnTxInterceptors)
		return f(ctx, wrappedStorage)
	}

	return itr.smStorageRepository.InTransaction(ctx, fWrapper)
}

func (itr *InterceptableTransactionalRepository) AddCreateAroundTxInterceptorProvider(objectType types.ObjectType, provider CreateAroundTxInterceptorProvider, order InterceptorOrder) {
	itr.validateCreateProviders(objectType, provider.Name(), order)
	itr.orderedCreateAroundTxProvidersNames[objectType] = insertName(itr.orderedCreateAroundTxProvidersNames[objectType], order.AroundTxPosition, provider.Name())
	if itr.createAroundTxProviders[objectType] == nil {
		itr.createAroundTxProviders[objectType] = make(map[string]OrderedCreateAroundTxInterceptorProvider)
	}
	itr.createAroundTxProviders[objectType][provider.Name()] = OrderedCreateAroundTxInterceptorProvider{
		InterceptorOrder:                  order,
		CreateAroundTxInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddCreateOnTxInterceptorProvider(objectType types.ObjectType, provider CreateOnTxInterceptorProvider, order InterceptorOrder) {
	itr.validateCreateProviders(objectType, provider.Name(), order)
	itr.orderedCreateOnTxProvidersNames[objectType] = insertName(itr.orderedCreateOnTxProvidersNames[objectType], order.OnTxPosition, provider.Name())
	if itr.createOnTxProviders[objectType] == nil {
		itr.createOnTxProviders[objectType] = make(map[string]OrderedCreateOnTxInterceptorProvider)
	}
	itr.createOnTxProviders[objectType][provider.Name()] = OrderedCreateOnTxInterceptorProvider{
		InterceptorOrder:              order,
		CreateOnTxInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddCreateInterceptorProvider(objectType types.ObjectType, provider CreateInterceptorProvider, order InterceptorOrder) {
	itr.validateCreateProviders(objectType, provider.Name(), order)
	itr.orderedCreateAroundTxProvidersNames[objectType] = insertName(itr.orderedCreateAroundTxProvidersNames[objectType], order.AroundTxPosition, provider.Name())
	itr.orderedCreateOnTxProvidersNames[objectType] = insertName(itr.orderedCreateOnTxProvidersNames[objectType], order.OnTxPosition, provider.Name())
	if itr.createProviders[objectType] == nil {
		itr.createProviders[objectType] = make(map[string]OrderedCreateInterceptorProvider)
	}
	itr.createProviders[objectType][provider.Name()] = OrderedCreateInterceptorProvider{
		InterceptorOrder:          order,
		CreateInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddUpdateAroundTxInterceptorProvider(objectType types.ObjectType, provider UpdateAroundTxInterceptorProvider, order InterceptorOrder) {
	itr.validateUpdateProviders(objectType, provider.Name(), order)
	itr.orderedUpdateAroundTxProvidersNames[objectType] = insertName(itr.orderedUpdateAroundTxProvidersNames[objectType], order.AroundTxPosition, provider.Name())
	if itr.updateAroundTxProviders[objectType] == nil {
		itr.updateAroundTxProviders[objectType] = make(map[string]OrderedUpdateAroundTxInterceptorProvider)
	}
	itr.updateAroundTxProviders[objectType][provider.Name()] = OrderedUpdateAroundTxInterceptorProvider{
		InterceptorOrder:                  order,
		UpdateAroundTxInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddUpdateOnTxInterceptorProvider(objectType types.ObjectType, provider UpdateOnTxInterceptorProvider, order InterceptorOrder) {
	itr.validateUpdateProviders(objectType, provider.Name(), order)
	itr.orderedUpdateOnTxProvidersNames[objectType] = insertName(itr.orderedUpdateOnTxProvidersNames[objectType], order.OnTxPosition, provider.Name())
	if itr.updateOnTxProviders[objectType] == nil {
		itr.updateOnTxProviders[objectType] = make(map[string]OrderedUpdateOnTxInterceptorProvider)
	}
	itr.updateOnTxProviders[objectType][provider.Name()] = OrderedUpdateOnTxInterceptorProvider{
		InterceptorOrder:              order,
		UpdateOnTxInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddUpdateInterceptorProvider(objectType types.ObjectType, provider UpdateInterceptorProvider, order InterceptorOrder) {
	itr.validateUpdateProviders(objectType, provider.Name(), order)
	itr.orderedUpdateAroundTxProvidersNames[objectType] = insertName(itr.orderedUpdateAroundTxProvidersNames[objectType], order.AroundTxPosition, provider.Name())
	itr.orderedUpdateOnTxProvidersNames[objectType] = insertName(itr.orderedUpdateOnTxProvidersNames[objectType], order.OnTxPosition, provider.Name())
	if itr.updateProviders[objectType] == nil {
		itr.updateProviders[objectType] = make(map[string]OrderedUpdateInterceptorProvider)
	}
	itr.updateProviders[objectType][provider.Name()] = OrderedUpdateInterceptorProvider{
		InterceptorOrder:          order,
		UpdateInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddDeleteAroundTxInterceptorProvider(objectType types.ObjectType, provider DeleteAroundTxInterceptorProvider, order InterceptorOrder) {
	itr.validateDeleteProviders(objectType, provider.Name(), order)
	itr.orderedDeleteAroundTxProvidersNames[objectType] = insertName(itr.orderedDeleteAroundTxProvidersNames[objectType], order.AroundTxPosition, provider.Name())
	if itr.deleteAroundTxProviders[objectType] == nil {
		itr.deleteAroundTxProviders[objectType] = make(map[string]OrderedDeleteAroundTxInterceptorProvider)
	}
	itr.deleteAroundTxProviders[objectType][provider.Name()] = OrderedDeleteAroundTxInterceptorProvider{
		InterceptorOrder:                  order,
		DeleteAroundTxInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddDeleteOnTxInterceptorProvider(objectType types.ObjectType, provider DeleteOnTxInterceptorProvider, order InterceptorOrder) {
	itr.validateDeleteProviders(objectType, provider.Name(), order)
	itr.orderedDeleteOnTxProvidersNames[objectType] = insertName(itr.orderedDeleteOnTxProvidersNames[objectType], order.OnTxPosition, provider.Name())
	if itr.deleteOnTxProviders[objectType] == nil {
		itr.deleteOnTxProviders[objectType] = make(map[string]OrderedDeleteOnTxInterceptorProvider)
	}
	itr.deleteOnTxProviders[objectType][provider.Name()] = OrderedDeleteOnTxInterceptorProvider{
		InterceptorOrder:              order,
		DeleteOnTxInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) AddDeleteInterceptorProvider(objectType types.ObjectType, provider DeleteInterceptorProvider, order InterceptorOrder) {
	itr.validateDeleteProviders(objectType, provider.Name(), order)
	itr.orderedDeleteAroundTxProvidersNames[objectType] = insertName(itr.orderedDeleteAroundTxProvidersNames[objectType], order.AroundTxPosition, provider.Name())
	itr.orderedDeleteOnTxProvidersNames[objectType] = insertName(itr.orderedDeleteOnTxProvidersNames[objectType], order.OnTxPosition, provider.Name())
	if itr.deleteProviders[objectType] == nil {
		itr.deleteProviders[objectType] = make(map[string]OrderedDeleteInterceptorProvider)
	}
	itr.deleteProviders[objectType][provider.Name()] = OrderedDeleteInterceptorProvider{
		InterceptorOrder:          order,
		DeleteInterceptorProvider: provider,
	}
}

func (itr *InterceptableTransactionalRepository) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors := itr.provideInterceptors()

	onTxInterceptorChain := func(ctx context.Context, obj types.Object) (types.Object, error) {
		var createdObj types.Object
		var err error

		if err := itr.smStorageRepository.InTransaction(ctx, func(ctx context.Context, txStorage Repository) error {
			interceptableRepository := newScopedRepositoryWithInterceptors(txStorage, providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors)
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

	var err error
	objectType := obj.GetType()
	if providedCreateInterceptors[objectType] != nil {
		obj, err = providedCreateInterceptors[objectType].AroundTxCreate(onTxInterceptorChain)(ctx, obj)
	} else {
		obj, err = onTxInterceptorChain(ctx, obj)
	}

	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (itr *InterceptableTransactionalRepository) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
	object, err := itr.smStorageRepository.Get(ctx, objectType, criteria...)
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

func (itr *InterceptableTransactionalRepository) Count(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (int, error) {
	return itr.smStorageRepository.Count(ctx, objectType, criteria...)
}

func (itr *InterceptableTransactionalRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors := itr.provideInterceptors()

	finalInterceptor := func(ctx context.Context, criteria ...query.Criterion) (types.ObjectList, error) {
		var result types.ObjectList
		var err error

		if err := itr.smStorageRepository.InTransaction(ctx, func(ctx context.Context, txStorage Repository) error {
			interceptableRepository := newScopedRepositoryWithInterceptors(txStorage, providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors)
			result, err = interceptableRepository.Delete(ctx, objectType, criteria...)
			if err != nil {
				return err
			}

			return nil
		}); err != nil {
			return nil, err
		}
		return result, nil
	}

	var objectList types.ObjectList
	var err error

	if providedDeleteInterceptors[objectType] != nil {
		objectList, err = providedDeleteInterceptors[objectType].AroundTxDelete(finalInterceptor)(ctx, criteria...)
	} else {
		objectList, err = finalInterceptor(ctx, criteria...)
	}

	if err != nil {
		return nil, err
	}

	return objectList, err
}

func (itr *InterceptableTransactionalRepository) Update(ctx context.Context, obj types.Object, labelChanges query.LabelChanges, criteria ...query.Criterion) (types.Object, error) {
	providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors := itr.provideInterceptors()

	finalInterceptor := func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		var result types.Object
		var err error

		if err = itr.smStorageRepository.InTransaction(ctx, func(ctx context.Context, txStorage Repository) error {
			interceptableRepository := newScopedRepositoryWithInterceptors(txStorage, providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors)
			result, err = interceptableRepository.Update(ctx, obj, labelChanges, criteria...)
			if err != nil {
				return err
			}

			return nil
		}); err != nil {
			return nil, err
		}

		return result, nil
	}

	var err error
	objectType := obj.GetType()
	if providedUpdateInterceptors[objectType] != nil {
		obj, err = providedUpdateInterceptors[objectType].AroundTxUpdate(finalInterceptor)(ctx, obj, labelChanges...)
	} else {
		obj, err = finalInterceptor(ctx, obj, labelChanges...)
	}

	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (itr *InterceptableTransactionalRepository) validateCreateProviders(objectType types.ObjectType, providerName string, order InterceptorOrder) {
	var existingProviderNames []string
	for _, existingProvider := range itr.createAroundTxProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}
	for _, existingProvider := range itr.createOnTxProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}
	for _, existingProvider := range itr.createProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}

	validateProviderOrder(order, existingProviderNames, providerName)
}

func (itr *InterceptableTransactionalRepository) validateUpdateProviders(objectType types.ObjectType, providerName string, order InterceptorOrder) {
	var existingProviderNames []string
	for _, existingProvider := range itr.updateAroundTxProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}
	for _, existingProvider := range itr.updateOnTxProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}
	for _, existingProvider := range itr.updateProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}

	validateProviderOrder(order, existingProviderNames, providerName)
}

func (itr *InterceptableTransactionalRepository) validateDeleteProviders(objectType types.ObjectType, providerName string, order InterceptorOrder) {
	var existingProviderNames []string
	for _, existingProvider := range itr.deleteAroundTxProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}
	for _, existingProvider := range itr.deleteOnTxProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}
	for _, existingProvider := range itr.deleteProviders[objectType] {
		existingProviderNames = append(existingProviderNames, existingProvider.Name())
	}

	validateProviderOrder(order, existingProviderNames, providerName)
}

func validateProviderOrder(order InterceptorOrder, existingProviderNames []string, providerName string) {
	if providerWithNameExists(existingProviderNames, providerName) {
		log.D().Panicf("%s create interceptor provider is already registered", providerName)
	}

	positionAroundTx, aroundTxName := order.AroundTxPosition.PositionType, order.AroundTxPosition.Name
	if positionAroundTx != PositionNone {
		if !providerWithNameExists(existingProviderNames, aroundTxName) {
			log.D().Panicf("could not find interceptor with name %s", aroundTxName)
		}
	}

	positionTx, nameTx := order.OnTxPosition.PositionType, order.OnTxPosition.Name
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
	createObjectTypes, updateObjectTypes, deleteObjectTypes := itr.mergeObjectTypes()

	providedCreateInterceptors := make(map[types.ObjectType]CreateInterceptor)
	for _, objectType := range createObjectTypes {
		providedCreateInterceptors[objectType] = itr.newCreateInterceptorChain(objectType)
	}
	providedUpdateInterceptors := make(map[types.ObjectType]UpdateInterceptor)
	for _, objectType := range updateObjectTypes {
		providedUpdateInterceptors[objectType] = itr.newUpdateInterceptorChain(objectType)
	}
	providedDeleteInterceptors := make(map[types.ObjectType]DeleteInterceptor)
	for _, objectType := range deleteObjectTypes {
		providedDeleteInterceptors[objectType] = itr.newDeleteInterceptorChain(objectType)
	}
	return providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors
}

func (itr *InterceptableTransactionalRepository) mergeObjectTypes() ([]types.ObjectType, []types.ObjectType, []types.ObjectType) {
	createObjectTypes := make([]types.ObjectType, 0)
	for objectType := range itr.orderedCreateAroundTxProvidersNames {
		createObjectTypes = append(createObjectTypes, objectType)
	}
	for objectType := range itr.orderedCreateOnTxProvidersNames {
		if _, ok := itr.orderedCreateAroundTxProvidersNames[objectType]; !ok {
			createObjectTypes = append(createObjectTypes, objectType)
		}
	}

	updateObjectTypes := make([]types.ObjectType, 0)
	for objectType := range itr.orderedUpdateAroundTxProvidersNames {
		updateObjectTypes = append(updateObjectTypes, objectType)
	}
	for objectType := range itr.orderedUpdateOnTxProvidersNames {
		if _, ok := itr.orderedUpdateAroundTxProvidersNames[objectType]; !ok {
			updateObjectTypes = append(updateObjectTypes, objectType)
		}
	}

	deleteObjectTypes := make([]types.ObjectType, 0)
	for objectType := range itr.orderedDeleteAroundTxProvidersNames {
		deleteObjectTypes = append(deleteObjectTypes, objectType)
	}
	for objectType := range itr.orderedDeleteOnTxProvidersNames {
		if _, ok := itr.orderedDeleteAroundTxProvidersNames[objectType]; !ok {
			deleteObjectTypes = append(deleteObjectTypes, objectType)
		}
	}
	return createObjectTypes, updateObjectTypes, deleteObjectTypes
}

func (itr *InterceptableTransactionalRepository) provideOnTxInterceptors() (
	map[types.ObjectType]func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc,
	map[types.ObjectType]func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc,
	map[types.ObjectType]func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc,
) {
	providedCreateInterceptors := make(map[types.ObjectType]func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc)
	for objectType := range itr.createOnTxProviders {
		providedCreateInterceptors[objectType] = itr.newCreateOnTxInterceptorChain(objectType).OnTxCreate
	}
	providedUpdateInterceptors := make(map[types.ObjectType]func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc)
	for objectType := range itr.updateOnTxProviders {
		providedUpdateInterceptors[objectType] = itr.newUpdateOnTxInterceptorChain(objectType).OnTxUpdate

	}
	providedDeleteInterceptors := make(map[types.ObjectType]func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc)
	for objectType := range itr.deleteOnTxProviders {
		providedDeleteInterceptors[objectType] = itr.newDeleteOnTxInterceptorChain(objectType).OnTxDelete
	}
	return providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors
}

// PositionType could be "before", "after" or "none"
type PositionType int

const (
	// PositionNone states that a position is not set and the item will be appended
	PositionNone PositionType = iota

	// PositionBefore states that a position should be calculated before another position
	PositionBefore

	// PositionAfter states that a position should be calculated after another position
	PositionAfter
)

type InterceptorPosition struct {
	PositionType PositionType
	Name         string
}

type InterceptorOrder struct {
	OnTxPosition     InterceptorPosition
	AroundTxPosition InterceptorPosition
}

// insertName inserts the given newInterceptorName into it's the expected position.
// The resulting names slice can then be used to wrap all interceptors into the right order
func insertName(names []string, position InterceptorPosition, newInterceptorName string) []string {
	if position.PositionType == PositionNone {
		names = append(names, newInterceptorName)
		return names
	}
	pos := findName(names, position.Name)
	if pos == -1 {
		panic(fmt.Errorf("could not find create API hook with name %s", position.Name))
	}
	names = append(names, "")
	if position.PositionType == PositionAfter {
		pos = pos + 1
	}
	copy(names[pos+1:], names[pos:])
	names[pos] = newInterceptorName
	return names
}

func findName(names []string, existingInterceptorName string) int {
	for i, name := range names {
		if name == existingInterceptorName {
			return i
		}
	}

	return -1
}
