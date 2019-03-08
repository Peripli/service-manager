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

type Hookable interface {
}

type Controller struct {
	PathParamID               string
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
				Path:   fmt.Sprintf("%s/{id}", c.ResourceBaseURL),
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
				Path:   fmt.Sprintf("%s/{id}", c.ResourceBaseURL),
			},
			Handler: c.DeleteSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   fmt.Sprintf("%s/{id}", c.ResourceBaseURL),
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
	onAPI := createHook.OnAPI(func(ctx context.Context, obj types.Object) (object types.Object, e error) {
		return obj, nil
	})
	result, err = onAPI(ctx, result)
	if err != nil {
		return nil, err
	}

	result.SetID(UUID.String())
	currentTime := time.Now().UTC()
	result.SetCreatedAt(currentTime)
	result.SetUpdatedAt(currentTime)

	onTransaction := createHook.OnTransaction(func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object) error {
		id, err := txStorage.Create(ctx, newObject)
		if err != nil {
			return util.HandleStorageError(err, string(c.ObjectType))
		}
		newObject.SetID(id)
		return nil
	})

	err = c.Repository.InTransaction(ctx, func(ctx context.Context, txStorage storage.Warehouse) error {
		return onTransaction(ctx, txStorage, result)
	})
	if err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusCreated, result)
}

func (c *Controller) DeleteObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %ss...", c.ObjectType)
	deleteHook := c.DeleteInterceptorProvider()
	onAPI := deleteHook.OnAPI(func(ctx context.Context, deleteionCriteria ...query.Criterion) error {
		return nil
	})
	criteria := query.CriteriaForContext(ctx)
	if err := onAPI(ctx, criteria...); err != nil {
		return nil, err
	}

	transactionOperation := deleteHook.OnTransaction(func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		return c.Repository.Delete(ctx, c.ObjectType, deletionCriteria...)
	})

	err := c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		_, err := transactionOperation(ctx, storage, criteria...)
		return util.HandleSelectionError(err, string(c.ObjectType))
	})

	if err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *Controller) DeleteSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[c.PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %s with id %s", c.ObjectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)

	deleteHook := c.DeleteInterceptorProvider()
	apiCall := deleteHook.OnAPI(func(ctx context.Context, deletionCriteria ...query.Criterion) error {
		return nil
	})
	if err := apiCall(ctx, byID); err != nil {
		return &web.Response{}, err
	}
	transactionOperation := deleteHook.OnTransaction(func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		return c.Repository.Delete(ctx, c.ObjectType, deletionCriteria...)
	})
	err := c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		_, err := transactionOperation(ctx, storage, byID)
		return util.HandleStorageError(err, string(c.ObjectType))
	})
	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, map[string]int{})
}

func (c *Controller) GetSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[c.PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting %s with id %s", c.ObjectType, objectID)

	object, err := c.Repository.Get(ctx, objectID, c.ObjectType)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.ObjectType))
	}
	object.SetCredentials(nil)
	return util.NewJSONResponse(http.StatusOK, object)
}

func (c *Controller) ListObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Getting all %ss", c.ObjectType)
	objects, err := c.Repository.List(ctx, c.ObjectType, query.CriteriaForContext(ctx)...)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}

	for i := 0; i < objects.Len(); i++ {
		obj := objects.ItemAt(i)
		obj.SetCredentials(nil)
	}

	return util.NewJSONResponse(http.StatusOK, objects)
}

func (c *Controller) PatchObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[c.PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating %s with id %s", c.ObjectType, objectID)

	result, err := c.Repository.Get(ctx, objectID, c.ObjectType)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.ObjectType))
	}
	createdAt := result.GetCreatedAt()

	changes, err := query.LabelChangesFromJSON(r.Body)
	if err != nil {
		return nil, util.HandleLabelChangeError(err)
	}
	if r.Body, err = sjson.DeleteBytes(r.Body, "labels"); err != nil {
		return nil, err
	}

	updateHook := c.UpdateInterceptorProvider()
	apiCall := updateHook.OnAPI(func(ctx context.Context, modifiedObject types.Object) (object types.Object, e error) {
		if err := util.BytesToObject(r.Body, modifiedObject); err != nil {
			return nil, err
		}
		// do not allow overriding these values via the API
		modifiedObject.SetID(objectID)
		modifiedObject.SetCreatedAt(createdAt)
		modifiedObject.SetUpdatedAt(time.Now().UTC())

		return modifiedObject, nil
	})

	object, err := apiCall(ctx, result)
	if err != nil {
		return nil, err
	}

	transactionOp := updateHook.OnTransaction(func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object, changes ...*query.LabelChange) error {
		var err error
		object, err = txStorage.Update(ctx, newObject, changes...)
		return err
	})

	if err = c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		return transactionOp(ctx, storage, object, changes...)
	}); err != nil {
		return nil, err
	}

	object.SetCredentials(nil)
	return util.NewJSONResponse(http.StatusOK, object)
}
