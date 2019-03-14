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

package base

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tidwall/sjson"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/extension"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const PathParamID = "id"

type Controller struct {
	resourceBaseURL           string
	objectType                types.ObjectType
	repository                storage.Repository
	objectBlueprint           func() types.Object
	CreateInterceptorProvider extension.CreateInterceptorProvider
	UpdateInterceptorProvider extension.UpdateInterceptorProvider
	DeleteInterceptorProvider extension.DeleteInterceptorProvider
}

func (c *Controller) InterceptsType() types.ObjectType {
	return c.objectType
}

func (c *Controller) AddCreateInterceptorProviders(providers ...extension.CreateInterceptorProvider) {
	if c.CreateInterceptorProvider == nil {
		c.CreateInterceptorProvider = extension.UnionCreateInterceptor(providers)
	} else {

		c.CreateInterceptorProvider = extension.UnionCreateInterceptor(append([]extension.CreateInterceptorProvider{c.CreateInterceptorProvider}, providers...))
	}
}

func (c *Controller) AddUpdateInterceptorProviders(providers ...extension.UpdateInterceptorProvider) {
	if c.UpdateInterceptorProvider == nil {
		c.UpdateInterceptorProvider = extension.UnionUpdateInterceptor(providers)
	} else {
		c.UpdateInterceptorProvider = extension.UnionUpdateInterceptor(append([]extension.UpdateInterceptorProvider{c.UpdateInterceptorProvider}, providers...))
	}
}

func (c *Controller) AddDeleteInterceptorProviders(providers ...extension.DeleteInterceptorProvider) {
	if c.DeleteInterceptorProvider == nil {
		c.DeleteInterceptorProvider = extension.UnionDeleteInterceptor(providers)
	} else {
		c.DeleteInterceptorProvider = extension.UnionDeleteInterceptor(append([]extension.DeleteInterceptorProvider{c.DeleteInterceptorProvider}, providers...))
	}
}

func NewController(repository storage.Repository, resourceBaseURL string, objectBlueprint func() types.Object) *Controller {
	return &Controller{
		repository:      repository,
		resourceBaseURL: resourceBaseURL,
		objectBlueprint: objectBlueprint,
		objectType:      objectBlueprint().GetType(),
	}
}

func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   c.resourceBaseURL,
			},
			Handler: c.CreateObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, PathParamID),
			},
			Handler: c.GetSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   c.resourceBaseURL,
			},
			Handler: c.ListObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   c.resourceBaseURL,
			},
			Handler: c.DeleteObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, PathParamID),
			},
			Handler: c.DeleteSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, PathParamID),
			},
			Handler: c.PatchObject,
		},
	}
}

func (c *Controller) CreateObject(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Creating new %s", c.objectType)

	createHook := c.CreateInterceptorProvider()

	onTransaction := func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object) error {
		id, err := txStorage.Create(ctx, newObject)
		if err != nil {
			return util.HandleStorageError(err, string(c.objectType))
		}
		newObject.SetID(id)
		return nil
	}
	if createHook != nil {
		onTransaction = createHook.OnTransaction(onTransaction)
	}

	onAPI := func(ctx context.Context, obj types.Object) (types.Object, error) {
		if err := c.repository.InTransaction(ctx, func(ctx context.Context, txStorage storage.Warehouse) error {
			return onTransaction(ctx, txStorage, obj)
		}); err != nil {
			return nil, err
		}
		return obj, nil
	}
	if createHook != nil {
		onAPI = createHook.OnAPI(onAPI)
	}
	result := c.objectBlueprint()
	if err := util.BytesToObject(r.Body, result); err != nil {
		return nil, err
	}

	if result.GetID() == "" {
		UUID, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("could not generate GUID for %s: %s", c.objectType, err)
		}
		result.SetID(UUID.String())
	}
	currentTime := time.Now().UTC()
	result.SetCreatedAt(currentTime)
	result.SetUpdatedAt(currentTime)

	result, err := onAPI(ctx, result)
	if err != nil {
		return nil, err
	}
	return web.NewJSONResponse(http.StatusCreated, result)
}

func (c *Controller) DeleteObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %ss...", c.objectType)
	deleteHook := c.DeleteInterceptorProvider()

	transactionOperation := func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		return c.repository.Delete(ctx, c.objectType, deletionCriteria...)
	}
	if deleteHook != nil {
		transactionOperation = deleteHook.OnTransaction(transactionOperation)
	}

	apiOperation := func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		var result types.ObjectList
		if err := c.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
			var err error
			result, err = transactionOperation(ctx, storage, deletionCriteria...)
			return util.HandleSelectionError(err, string(c.objectType))
		}); err != nil {
			return nil, err
		}
		return result, nil
	}
	if deleteHook != nil {
		apiOperation = deleteHook.OnAPI(apiOperation)
	}
	criteria := query.CriteriaForContext(ctx)
	if _, err := apiOperation(ctx, criteria...); err != nil {
		return nil, err
	}

	return web.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *Controller) DeleteSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %s with id %s", c.objectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)
	ctx, err := query.AddCriteria(r.Context(), byID)
	if err != nil {
		return nil, err
	}
	r.Request = r.WithContext(ctx)
	return c.DeleteObjects(r)
}

func (c *Controller) GetSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting %s with id %s", c.objectType, objectID)

	object, err := c.repository.Get(ctx, c.objectType, objectID)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.objectType))
	}
	if secured, ok := object.(types.Secured); ok {
		secured.SetCredentials(nil)
	} else {
		log.C(ctx).Debugf("Object of type %s with id %s is not secured, so no credentials are cleaned up on response", object.GetType(), object.GetID())
	}
	return web.NewJSONResponse(http.StatusOK, object)
}

func (c *Controller) ListObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Getting all %ss", c.objectType)
	objectList, err := c.repository.List(ctx, c.objectType, query.CriteriaForContext(ctx)...)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}
	for i := 0; i < objectList.Len(); i++ {
		obj := objectList.ItemAt(i)
		if secured, ok := obj.(types.Secured); ok {
			secured.SetCredentials(nil)
		} else {
			log.C(ctx).Debugf("Object of type %s with id %s is not secured, so no credentials are cleaned up on response", obj.GetType(), obj.GetID())
		}
	}

	return web.NewJSONResponse(http.StatusOK, objectList)
}

func (c *Controller) PatchObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating %s with id %s", c.objectType, objectID)

	labelChanges, err := query.LabelChangesFromJSON(r.Body)
	if err != nil {
		return nil, err
	}
	if r.Body, err = sjson.DeleteBytes(r.Body, "labels"); err != nil {
		return nil, err
	}

	objectChanges := extension.UpdateContext{
		LabelChanges:  labelChanges,
		ObjectChanges: r.Body,
		ObjectID:      objectID,
	}

	updateHook := c.UpdateInterceptorProvider()
	transactionOp := updateHook.OnTransaction(func(ctx context.Context, txStorage storage.Warehouse, obj types.Object, updateChanges extension.UpdateContext) (types.Object, error) {
		createdAt := obj.GetCreatedAt()
		if err := util.BytesToObject(updateChanges.ObjectChanges, obj); err != nil {
			return nil, err
		}
		obj.SetID(objectID)
		obj.SetCreatedAt(createdAt)
		obj.SetUpdatedAt(time.Now().UTC())
		return txStorage.Update(ctx, obj, updateChanges.LabelChanges...)
	})
	if updateHook != nil {
		transactionOp = updateHook.OnTransaction(transactionOp)
	}

	apiOperation := func(ctx context.Context, updateChanges extension.UpdateContext) (types.Object, error) {
		var result types.Object
		if err = c.repository.InTransaction(ctx, func(ctx context.Context, txStorage storage.Warehouse) error {
			oldObject, err := txStorage.Get(ctx, c.objectType, objectID)
			if err != nil {
				return util.HandleStorageError(err, string(c.objectType))
			}
			result, err = transactionOp(ctx, txStorage, oldObject, updateChanges)
			return err
		}); err != nil {
			return nil, err
		}
		return result, nil
	}
	if updateHook != nil {
		apiOperation = updateHook.OnAPI(apiOperation)
	}

	object, err := apiOperation(ctx, objectChanges)
	if err != nil {
		return nil, err
	}
	if obj, ok := object.(types.Secured); ok {
		obj.SetCredentials(nil)
	} else {
		log.C(ctx).Debugf("Object of type %s with id %s is not secured, so no credentials are cleaned up on response", object.GetType(), object.GetID())
	}
	return web.NewJSONResponse(http.StatusOK, object)
}
