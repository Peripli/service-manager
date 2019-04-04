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

func NewInterceptableRepository(repository Repository, encrypter security.Encrypter) *InterceptableRepository {
	return &InterceptableRepository{
		delegate:        repository,
		encrypter:       encrypter,
		createProviders: make(map[types.ObjectType][]CreateInterceptorProvider),
		updateProviders: make(map[types.ObjectType][]UpdateInterceptorProvider),
		deleteProviders: make(map[types.ObjectType][]DeleteInterceptorProvider),
	}
}

type InterceptableRepository struct {
	delegate        Repository
	encrypter       security.Encrypter
	createProviders map[types.ObjectType][]CreateInterceptorProvider
	updateProviders map[types.ObjectType][]UpdateInterceptorProvider
	deleteProviders map[types.ObjectType][]DeleteInterceptorProvider
}

func (ir *InterceptableRepository) AddCreateInterceptorProviders(objectType types.ObjectType, providers ...CreateInterceptorProvider) {
	ir.validateCreateProviders(objectType, providers)
	ir.createProviders[objectType] = append(ir.createProviders[objectType], providers...)
}

func (ir *InterceptableRepository) AddUpdateInterceptorProviders(objectType types.ObjectType, providers ...UpdateInterceptorProvider) {
	ir.validateUpdateProviders(objectType, providers)
	ir.updateProviders[objectType] = append(ir.updateProviders[objectType], providers...)
}

func (ir *InterceptableRepository) AddDeleteInterceptorProviders(objectType types.ObjectType, providers ...DeleteInterceptorProvider) {
	ir.deleteProviders[objectType] = append(ir.deleteProviders[objectType], providers...)
}

type finalCreateObjectInterceptor struct {
	encrypter  security.Encrypter
	repository Repository
	objectType types.ObjectType
	txChain    func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc
}

func (final *finalCreateObjectInterceptor) InterceptCreateOnTx(ctx context.Context, txStorage Warehouse, newObject types.Object) error {
	id, err := txStorage.Create(ctx, newObject)
	if err != nil {
		return err
	}
	newObject.SetID(id)
	return nil
}

func (final *finalCreateObjectInterceptor) InterceptCreateAroundTx(ctx context.Context, obj types.Object) (types.Object, error) {
	if err := transformCredentials(ctx, obj, final.encrypter.Encrypt); err != nil {
		return nil, err
	}
	if err := final.repository.InTransaction(ctx, func(ctx context.Context, storage Warehouse) error {
		if err := final.txChain(final.InterceptCreateOnTx)(ctx, storage, obj); err != nil {
			return util.HandleSelectionError(err, string(final.objectType))
		}
		if securedObj, isSecured := obj.(types.Secured); isSecured {
			securedObj.SetCredentials(nil)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return obj, nil
}

func (ir *InterceptableRepository) Create(ctx context.Context, obj types.Object) (string, error) {
	objectType := obj.GetType()
	interceptorsChain := NewCreateInterceptorChain(ir.createProviders[objectType])

	finalInterceptor := &finalCreateObjectInterceptor{
		encrypter:  ir.encrypter,
		objectType: objectType,
		repository: ir.delegate,
		txChain:    interceptorsChain.OnTxCreate,
	}

	obj, err := interceptorsChain.AroundTxCreate(finalInterceptor.InterceptCreateAroundTx)(ctx, obj)
	if err != nil {
		return "", err
	}
	return obj.GetID(), nil
}

func (ir *InterceptableRepository) Get(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
	object, err := ir.delegate.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}
	if err = transformCredentials(ctx, object, ir.encrypter.Decrypt); err != nil {
		return nil, err
	}
	return object, nil
}

func (ir *InterceptableRepository) List(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	objectList, err := ir.delegate.List(ctx, objectType, criteria...)
	if err != nil {
		return nil, err
	}
	// TODO: There is no usecase for decrypting credentials on List operation,
	// 		 because of the decrypter Postgres runs out of connections as each decrypt operations
	// 		 calls the db to get the decryption key

	// for i := 0; i < objectList.Len(); i++ {
	// 	obj := objectList.ItemAt(i)
	// 	if err = transformCredentials(ctx, obj, ir.encrypter.Decrypt); err != nil {
	// 		return err
	// 	}
	// }

	return objectList, nil
}

type finalDeleteObjectInterceptor struct {
	repository Repository
	objectType types.ObjectType
	txChain    func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc
}

func (final *finalDeleteObjectInterceptor) InterceptDeleteOnTx(ctx context.Context, txStorage Warehouse, criteria ...query.Criterion) (types.ObjectList, error) {
	return txStorage.Delete(ctx, final.objectType, criteria...)
}

func (final *finalDeleteObjectInterceptor) InterceptDeleteAroundTx(ctx context.Context, criteria ...query.Criterion) (types.ObjectList, error) {
	var result types.ObjectList
	if err := final.repository.InTransaction(ctx, func(ctx context.Context, txStorage Warehouse) error {
		var err error
		result, err = final.txChain(final.InterceptDeleteOnTx)(ctx, txStorage, criteria...)
		if err != nil {
			return util.HandleSelectionError(err, string(final.objectType))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (ir *InterceptableRepository) Delete(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.ObjectList, error) {
	interceptorsChain := NewDeleteInterceptorChain(ir.deleteProviders[objectType])

	finalInterceptor := &finalDeleteObjectInterceptor{
		repository: ir.delegate,
		objectType: objectType,
		txChain:    interceptorsChain.OnTxDelete,
	}
	// TODO the chain itself is also a DeleteInterceptor - not sure if we can somehow leverage this. Ideas?
	return interceptorsChain.AroundTxDelete(finalInterceptor.InterceptDeleteAroundTx)(ctx, criteria...)
}

type finalUpdateObjectInterceptor struct {
	encrypter  security.Encrypter
	repository Repository
	objectType types.ObjectType
	txChain    func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc
}

func (final *finalUpdateObjectInterceptor) InterceptUpdateOnTxFunc(ctx context.Context, txStorage Warehouse, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	return txStorage.Update(ctx, obj, labelChanges...)
}

func (final *finalUpdateObjectInterceptor) InterceptUpdateAroundTxInterceptUpdateAroundTx(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	if err := transformCredentials(ctx, obj, final.encrypter.Encrypt); err != nil {
		return nil, err
	}
	var result types.Object
	if err := final.repository.InTransaction(ctx, func(ctx context.Context, txStorage Warehouse) error {
		var err error
		result, err = final.txChain(final.InterceptUpdateOnTxFunc)(ctx, txStorage, obj, labelChanges...)
		return util.HandleStorageError(err, string(final.objectType))
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (ir *InterceptableRepository) Update(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	objectType := obj.GetType()
	interceptorsChain := NewUpdateInterceptorChain(ir.updateProviders[objectType])

	finalInterceptor := &finalUpdateObjectInterceptor{
		encrypter:  ir.encrypter,
		objectType: objectType,
		repository: ir.delegate,
		txChain:    interceptorsChain.OnTxUpdate,
	}

	object, err := interceptorsChain.AroundTxUpdate(finalInterceptor.InterceptUpdateAroundTxInterceptUpdateAroundTx)(ctx, obj, labelChanges...)
	if err != nil {
		return nil, err
	}
	if securedObj, isSecured := object.(types.Secured); isSecured {
		securedObj.SetCredentials(nil)
	}
	return object, nil
}

func (ir *InterceptableRepository) Credentials() Credentials {
	return ir.delegate.Credentials()
}

func (ir *InterceptableRepository) Security() Security {
	return ir.delegate.Security()
}

func (ir *InterceptableRepository) InTransaction(ctx context.Context, f func(ctx context.Context, storage Warehouse) error) error {
	return ir.delegate.InTransaction(ctx, f)
}

func (ir *InterceptableRepository) validateCreateProvidersNames(objectType types.ObjectType, name string) {
	found := false
	for _, p := range ir.createProviders[objectType] {
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

func (ir *InterceptableRepository) validateCreateProviders(objectType types.ObjectType, newProviders []CreateInterceptorProvider) {
	for _, newProvider := range newProviders {
		interceptor := newProvider.Provide()
		if ordered, ok := newProvider.(Ordered); ok {
			positionAPI, nameAPI := ordered.PositionAPI()
			positionTx, nameTx := ordered.PositionTx()
			if positionAPI != PositionNone {
				ir.validateCreateProvidersNames(objectType, nameAPI)
			}
			if positionTx != PositionNone {
				ir.validateCreateProvidersNames(objectType, nameTx)
			}
		}
		for _, p := range ir.createProviders[objectType] {
			if n, ok := p.(Named); ok {
				if n.Name() == interceptor.Name() {
					log.D().Panicf("%s create interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}

func (ir *InterceptableRepository) validateUpdateProvidersNames(objectType types.ObjectType, name string) {
	found := false
	for _, p := range ir.updateProviders[objectType] {
		if p.Provide().Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (ir *InterceptableRepository) validateUpdateProviders(objectType types.ObjectType, newProviders []UpdateInterceptorProvider) {
	for _, newProvider := range newProviders {
		interceptor := newProvider.Provide()
		if ordered, ok := newProvider.(Ordered); ok {
			positionAPI, nameAPI := ordered.PositionAPI()
			positionTx, nameTx := ordered.PositionTx()
			if positionAPI != PositionNone {
				ir.validateUpdateProvidersNames(objectType, nameAPI)
			}
			if positionTx != PositionNone {
				ir.validateUpdateProvidersNames(objectType, nameTx)
			}
		}
		for _, p := range ir.updateProviders[objectType] {
			if n, ok := p.(Named); ok {
				if n.Name() == interceptor.Name() {
					log.D().Panicf("%s update interceptor provider is already registered", n.Name())
				}
			}
		}
	}
}

func (ir *InterceptableRepository) validateDeleteProvidersNames(objectType types.ObjectType, name string) {
	found := false
	for _, p := range ir.deleteProviders[objectType] {
		if p.Provide().Name() == name {
			found = true
			break
		}
	}
	if !found {
		log.D().Panicf("could not find interceptor with name %s", name)
	}
}

func (ir *InterceptableRepository) validateDeleteProviders(objectType types.ObjectType, newProviders []DeleteInterceptorProvider) {
	for _, newProvider := range newProviders {
		if ordered, ok := newProvider.(Ordered); ok {
			positionAPI, nameAPI := ordered.PositionAPI()
			positionTx, nameTx := ordered.PositionTx()
			if positionAPI != PositionNone {
				ir.validateDeleteProvidersNames(objectType, nameAPI)
			}
			if positionTx != PositionNone {
				ir.validateDeleteProvidersNames(objectType, nameTx)
			}
		}
		for _, p := range ir.deleteProviders[objectType] {
			if n, ok := p.(Named); ok {
				if n.Name() == newProvider.Provide().Name() {
					log.D().Panicf("%s delete interceptor provider is already registered", n.Name())
				}
			}
		}
	}
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
