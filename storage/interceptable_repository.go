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

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
)

func NewInterceptableTransactionalRepository(repository TransactionalRepository, encrypter security.Encrypter) *InterceptableTransactionalRepository {
	return &InterceptableTransactionalRepository{
		encrypter:           encrypter,
		smStorageRepository: repository,
		createProviders:     make(map[types.ObjectType][]CreateInterceptorProvider),
		updateProviders:     make(map[types.ObjectType][]UpdateInterceptorProvider),
		deleteProviders:     make(map[types.ObjectType][]DeleteInterceptorProvider),
	}
}

func newInterceptableRepository(repository Repository,
	encrypter security.Encrypter,
	providedCreateInterceptors map[types.ObjectType]CreateInterceptor,
	providedUpdateInterceptors map[types.ObjectType]UpdateInterceptor,
	providedDeleteInterceptors map[types.ObjectType]DeleteInterceptor) *interceptableRepository {
	return &interceptableRepository{
		repositoryInTransaction: repository,
		encrypter:               encrypter,
		createInterceptor:       providedCreateInterceptors,
		updateInterceptor:       providedUpdateInterceptors,
		deleteInterceptor:       providedDeleteInterceptors,
	}
}

type InterceptableTransactionalRepository struct {
	encrypter security.Encrypter

	smStorageRepository TransactionalRepository

	createProviders map[types.ObjectType][]CreateInterceptorProvider
	updateProviders map[types.ObjectType][]UpdateInterceptorProvider
	deleteProviders map[types.ObjectType][]DeleteInterceptorProvider
}

type interceptableRepository struct {
	encrypter security.Encrypter

	repositoryInTransaction Repository

	createInterceptor map[types.ObjectType]CreateInterceptor
	updateInterceptor map[types.ObjectType]UpdateInterceptor
	deleteInterceptor map[types.ObjectType]DeleteInterceptor
}

func (ir *interceptableRepository) Create(ctx context.Context, obj types.Object) (string, error) {
	objectType := obj.GetType()
	if err := transformCredentials(ctx, obj, ir.encrypter.Encrypt); err != nil {
		return "", err
	}

	createObjectFunc := func(ctx context.Context, _ Repository, newObject types.Object) error {
		id, err := ir.repositoryInTransaction.Create(ctx, newObject)
		if err != nil {
			return util.HandleStorageError(err, string(objectType))
		}
		obj.SetID(id)

		return nil
	}

	var err error
	if _, found := ir.createInterceptor[objectType]; found {
		err = ir.createInterceptor[objectType].OnTxCreate(createObjectFunc)(ctx, ir, obj)
	} else {
		err = createObjectFunc(ctx, ir.repositoryInTransaction, obj)
	}

	if err != nil {
		return "", err
	}

	if securedObj, isSecured := obj.(types.Secured); isSecured {
		securedObj.SetCredentials(nil)
	}

	return obj.GetID(), nil
}

func (ir *interceptableRepository) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	object, err := ir.repositoryInTransaction.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}
	if err = transformCredentials(ctx, object, ir.encrypter.Decrypt); err != nil {
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
	deleteObjectFunc := func(ctx context.Context, _ Repository, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		objectList, err := ir.repositoryInTransaction.Delete(ctx, objectType, deletionCriteria...)
		if err != nil {
			return nil, util.HandleSelectionError(err, string(objectType))
		}

		return objectList, nil
	}

	var objectList types.ObjectList
	var err error

	if _, found := ir.deleteInterceptor[objectType]; found {
		objectList, err = ir.deleteInterceptor[objectType].OnTxDelete(deleteObjectFunc)(ctx, ir, criteria...)
	} else {
		objectList, err = deleteObjectFunc(ctx, ir.repositoryInTransaction, criteria...)
	}

	if err != nil {
		return nil, err
	}

	return objectList, nil
}

func (ir *interceptableRepository) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	objectType := obj.GetType()
	if err := transformCredentials(ctx, obj, ir.encrypter.Encrypt); err != nil {
		return nil, err
	}

	updateObjFunc := func(ctx context.Context, _ Repository, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		object, err := ir.repositoryInTransaction.Update(ctx, obj, labelChanges...)
		if err != nil {
			return nil, util.HandleStorageError(err, string(objectType))
		}

		return object, nil
	}

	var updatedObj types.Object
	var err error

	if _, found := ir.updateInterceptor[objectType]; found {
		updatedObj, err = ir.updateInterceptor[objectType].OnTxUpdate(updateObjFunc)(ctx, ir, obj, labelChanges...)
	} else {
		updatedObj, err = updateObjFunc(ctx, ir, obj, labelChanges...)
	}

	if err != nil {
		return nil, err
	}

	if securedObj, isSecured := updatedObj.(types.Secured); isSecured {
		securedObj.SetCredentials(nil)
	}

	return updatedObj, nil
}

func (itr *InterceptableTransactionalRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Repository) error) error {
	return itr.smStorageRepository.InTransaction(ctx, f)
}

func (itr *InterceptableTransactionalRepository) AddCreateInterceptorProviders(objectType types.ObjectType, providers ...CreateInterceptorProvider) {
	itr.validateCreateProviders(objectType, providers)
	itr.createProviders[objectType] = append(itr.createProviders[objectType], providers...)
}

func (itr *InterceptableTransactionalRepository) AddUpdateInterceptorProviders(objectType types.ObjectType, providers ...UpdateInterceptorProvider) {
	itr.validateUpdateProviders(objectType, providers)
	itr.updateProviders[objectType] = append(itr.updateProviders[objectType], providers...)
}

func (itr *InterceptableTransactionalRepository) AddDeleteInterceptorProviders(objectType types.ObjectType, providers ...DeleteInterceptorProvider) {
	itr.validateDeleteProviders(objectType, providers)
	itr.deleteProviders[objectType] = append(itr.deleteProviders[objectType], providers...)
}

type finalCreateObjectInterceptor struct {
	encrypter                  security.Encrypter
	repository                 TransactionalRepository
	objectType                 types.ObjectType
	providedCreateInterceptors map[types.ObjectType]CreateInterceptor
	providedUpdateInterceptors map[types.ObjectType]UpdateInterceptor
	providedDeleteInterceptors map[types.ObjectType]DeleteInterceptor
}

func (final *finalCreateObjectInterceptor) InterceptCreateOnTx(ctx context.Context, txStorage Repository, newObject types.Object) error {
	id, err := txStorage.Create(ctx, newObject)
	if err != nil {
		return err
	}
	newObject.SetID(id)
	return nil
}

func (final *finalCreateObjectInterceptor) InterceptCreateAroundTx(ctx context.Context, obj types.Object) (types.Object, error) {
	var id string
	var err error

	if err := final.repository.InTransaction(ctx, func(ctx context.Context, txStorage Repository) error {
		interceptableRepository := newInterceptableRepository(txStorage, final.encrypter, final.providedCreateInterceptors, final.providedUpdateInterceptors, final.providedDeleteInterceptors)
		id, err = interceptableRepository.Create(ctx, obj)
		if err != nil {
			return err
		}
		if securedObj, isSecured := obj.(types.Secured); isSecured {
			securedObj.SetCredentials(nil)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	obj.SetID(id)
	return obj, nil
}

func (itr *InterceptableTransactionalRepository) Create(ctx context.Context, obj types.Object) (string, error) {
	providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors := itr.provideInterceptors()

	objectType := obj.GetType()

	finalInterceptor := &finalCreateObjectInterceptor{
		encrypter:                  itr.encrypter,
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
		return "", err
	}

	return obj.GetID(), nil
}

func (itr *InterceptableTransactionalRepository) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	object, err := itr.smStorageRepository.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}
	if err = transformCredentials(ctx, object, itr.encrypter.Decrypt); err != nil {
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
	encrypter                  security.Encrypter
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
		interceptableRepository := newInterceptableRepository(txStorage, final.encrypter, final.providedCreateInterceptors, final.providedUpdateInterceptors, final.providedDeleteInterceptors)
		result, err = interceptableRepository.Delete(ctx, final.objectType, criteria...)
		if err != nil {
			return util.HandleSelectionError(err, string(final.objectType))
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
		encrypter:                  itr.encrypter,
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
	encrypter                  security.Encrypter
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
		interceptableRepository := newInterceptableRepository(txStorage, final.encrypter, final.providedCreateInterceptors, final.providedUpdateInterceptors, final.providedDeleteInterceptors)

		result, err = interceptableRepository.Update(ctx, obj, labelChanges...)
		if err != nil {
			return util.HandleStorageError(err, string(final.objectType))
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
		encrypter:                  itr.encrypter,
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

	if securedObj, isSecured := obj.(types.Secured); isSecured {
		securedObj.SetCredentials(nil)
	}

	return obj, nil
}

func (itr *InterceptableTransactionalRepository) Credentials() Credentials {
	return itr.smStorageRepository.Credentials()
}

func (itr *InterceptableTransactionalRepository) Security() Security {
	return itr.smStorageRepository.Security()
}
func (ir *interceptableRepository) Credentials() Credentials {
	return ir.repositoryInTransaction.Credentials()
}

func (ir *interceptableRepository) Security() Security {
	return ir.repositoryInTransaction.Security()
}

func transformCredentials(ctx context.Context, obj types.Object, transformationFunc func(context.Context, []byte) ([]byte, error)) error {
	securedObj, isSecured := obj.(types.Secured)
	if isSecured {
		credentials := securedObj.GetCredentials()
		if credentials != nil {
			transformedPassword, err := transformationFunc(ctx, []byte(credentials.Basic.Password))
			if err != nil {
				return err
			}
			credentials.Basic.Password = string(transformedPassword)
			securedObj.SetCredentials(credentials)
		}
	}
	return nil
}

func (itr *InterceptableTransactionalRepository) validateCreateProvidersNames(objectType types.ObjectType, name string) {
	found := false
	for _, p := range itr.createProviders[objectType] {
		interceptor := p.Provide()
		if interceptor.Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (itr *InterceptableTransactionalRepository) validateCreateProviders(objectType types.ObjectType, newProviders []CreateInterceptorProvider) {
	for _, newProvider := range newProviders {
		interceptor := newProvider.Provide()
		if ordered, ok := newProvider.(Ordered); ok {
			positionAroundTx, nameAPI := ordered.PositionAroundTx()
			positionTx, nameTx := ordered.PositionTx()
			if positionAroundTx != PositionNone {
				itr.validateCreateProvidersNames(objectType, nameAPI)
			}
			if positionTx != PositionNone {
				itr.validateCreateProvidersNames(objectType, nameTx)
			}
		}
		for _, p := range itr.createProviders[objectType] {
			currentInterceptor := p.Provide()
			if n, ok := currentInterceptor.(Named); ok {
				if n.Name() == interceptor.Name() {
					log.D().Panicf("%s create interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}

func (itr *InterceptableTransactionalRepository) validateUpdateProvidersNames(objectType types.ObjectType, name string) {
	found := false
	for _, p := range itr.updateProviders[objectType] {
		if p.Provide().Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (itr *InterceptableTransactionalRepository) validateUpdateProviders(objectType types.ObjectType, newProviders []UpdateInterceptorProvider) {
	for _, newProvider := range newProviders {
		interceptor := newProvider.Provide()
		if ordered, ok := newProvider.(Ordered); ok {
			positionAroundTx, nameAPI := ordered.PositionAroundTx()
			positionTx, nameTx := ordered.PositionTx()
			if positionAroundTx != PositionNone {
				itr.validateUpdateProvidersNames(objectType, nameAPI)
			}
			if positionTx != PositionNone {
				itr.validateUpdateProvidersNames(objectType, nameTx)
			}
		}
		for _, p := range itr.updateProviders[objectType] {
			currentInterceptor := p.Provide()
			if n, ok := currentInterceptor.(Named); ok {
				if n.Name() == interceptor.Name() {
					log.D().Panicf("%s update interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}

func (itr *InterceptableTransactionalRepository) validateDeleteProvidersNames(objectType types.ObjectType, name string) {
	found := false
	for _, p := range itr.deleteProviders[objectType] {
		if p.Provide().Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (itr *InterceptableTransactionalRepository) validateDeleteProviders(objectType types.ObjectType, newProviders []DeleteInterceptorProvider) {
	for _, newProvider := range newProviders {
		if ordered, ok := newProvider.(Ordered); ok {
			positionAroundTx, nameAPI := ordered.PositionAroundTx()
			positionTx, nameTx := ordered.PositionTx()
			if positionAroundTx != PositionNone {
				itr.validateDeleteProvidersNames(objectType, nameAPI)
			}
			if positionTx != PositionNone {
				itr.validateDeleteProvidersNames(objectType, nameTx)
			}
		}
		for _, p := range itr.deleteProviders[objectType] {
			currentInterceptor := p.Provide()
			if n, ok := currentInterceptor.(Named); ok {
				if n.Name() == newProvider.Provide().Name() {
					log.D().Panicf("%s delete interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}

func (itr *InterceptableTransactionalRepository) provideInterceptors() (map[types.ObjectType]CreateInterceptor, map[types.ObjectType]UpdateInterceptor, map[types.ObjectType]DeleteInterceptor) {
	providedCreateInterceptors := make(map[types.ObjectType]CreateInterceptor)
	for objectType, providers := range itr.createProviders {
		providedCreateInterceptors[objectType] = NewCreateInterceptorChain(providers)
	}
	providedUpdateInterceptors := make(map[types.ObjectType]UpdateInterceptor)
	for objectType, providers := range itr.updateProviders {
		providedUpdateInterceptors[objectType] = NewUpdateInterceptorChain(providers)

	}
	providedDeleteInterceptors := make(map[types.ObjectType]DeleteInterceptor)
	for objectType, providers := range itr.deleteProviders {
		providedDeleteInterceptors[objectType] = NewDeleteInterceptorChain(providers)
	}
	return providedCreateInterceptors, providedUpdateInterceptors, providedDeleteInterceptors
}
