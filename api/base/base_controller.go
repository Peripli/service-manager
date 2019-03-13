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

	"github.com/Peripli/service-manager/pkg/extension"
	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const pathParamID = "id"

type Controller struct {
	ResourceBaseURL           string
	ObjectType                types.ObjectType
	Repository                storage.Repository
	ObjectBlueprint           func() types.Object
	CreateInterceptorProvider extension.CreateInterceptorProvider
	UpdateInterceptorProvider extension.UpdateInterceptorProvider
	DeleteInterceptorProvider extension.DeleteInterceptorProvider
}

func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   c.ResourceBaseURL,
			},
			Handler: c.CreateObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}", c.ResourceBaseURL, pathParamID),
			},
			Handler: c.GetSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   c.ResourceBaseURL,
			},
			Handler: c.ListObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   c.ResourceBaseURL,
			},
			Handler: c.DeleteObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   fmt.Sprintf("%s/{%s}", c.ResourceBaseURL, pathParamID),
			},
			Handler: c.DeleteSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   fmt.Sprintf("%s/{%s}", c.ResourceBaseURL, pathParamID),
			},
			Handler: c.PatchObject,
		},
	}
}

func (c *Controller) CreateObject(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Creating new %s", c.ObjectType)
	result := c.ObjectBlueprint()
	if err := util.BytesToObject(r.Body, result); err != nil {
		return nil, err
	}
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", c.ObjectType, err)
	}
	createHook := c.CreateInterceptorProvider()

	onTransaction := func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object) error {
		id, err := txStorage.Create(ctx, newObject)
		if err != nil {
			return util.HandleStorageError(err, string(c.ObjectType))
		}
		newObject.SetID(id)
		return nil
	}
	if createHook != nil {
		onTransaction = createHook.OnTransaction(onTransaction)
	}

	onAPI := func(ctx context.Context, obj types.Object) (types.Object, error) {
		if err = c.Repository.InTransaction(ctx, func(ctx context.Context, txStorage storage.Warehouse) error {
			return onTransaction(ctx, txStorage, obj)
		}); err != nil {
			return nil, err
		}
		return obj, nil
	}
	if createHook != nil {
		onAPI = createHook.OnAPI(onAPI)
	}

	result.SetID(UUID.String())
	currentTime := time.Now().UTC()
	result.SetCreatedAt(currentTime)
	result.SetUpdatedAt(currentTime)

	result, err = onAPI(ctx, result)
	if err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusCreated, result)
}

func (c *Controller) DeleteObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %ss...", c.ObjectType)
	deleteHook := c.DeleteInterceptorProvider()

	transactionOperation := func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		return c.Repository.Delete(ctx, c.ObjectType, deletionCriteria...)
	}
	if deleteHook != nil {
		transactionOperation = deleteHook.OnTransaction(transactionOperation)
	}

	apiOperation := func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		var result types.ObjectList
		if err := c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
			var err error
			result, err = transactionOperation(ctx, storage, deletionCriteria...)
			return util.HandleSelectionError(err, string(c.ObjectType))
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

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *Controller) DeleteSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[pathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %s with id %s", c.ObjectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)
	ctx, err := query.AddCriteria(r.Context(), byID)
	if err != nil {
		return nil, err
	}
	r.Request = r.WithContext(ctx)
	return c.DeleteObjects(r)
}

func (c *Controller) GetSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[pathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting %s with id %s", c.ObjectType, objectID)

	object, err := c.Repository.Get(ctx, objectID, c.ObjectType)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.ObjectType))
	}
	if secured, ok := object.(types.Secured); ok {
		secured.SetCredentials(nil)
	} else {
		log.C(ctx).Debugf("Object of type %s with id %s is not secured, so no credentials are cleaned up on response", object.GetType(), object.GetID())
	}
	return util.NewJSONResponse(http.StatusOK, object)
}

func (c *Controller) ListObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Getting all %ss", c.ObjectType)
	objectList, err := c.Repository.List(ctx, c.ObjectType, query.CriteriaForContext(ctx)...)
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

	return util.NewJSONResponse(http.StatusOK, objectList)
}

func (c *Controller) PatchObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[pathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating %s with id %s", c.ObjectType, objectID)

	labelChanges, err := query.LabelChangesFromJSON(r.Body)
	if err != nil {
		return nil, util.HandleLabelChangeError(err)
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
		if err = c.Repository.InTransaction(ctx, func(ctx context.Context, txStorage storage.Warehouse) error {
			oldObject, err := txStorage.Get(ctx, objectID, c.ObjectType)
			if err != nil {
				return util.HandleStorageError(err, string(c.ObjectType))
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
	return util.NewJSONResponse(http.StatusOK, object)
}
