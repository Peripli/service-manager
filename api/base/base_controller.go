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

type Controller struct {
	PathParamID     string
	ObjectType      types.ObjectType
	Repository      storage.Repository
	ObjectBlueprint func() types.Object
	CreateHookFunc  extension.CreateHookFunc
	UpdateHookFunc  extension.UpdateHookFunc
	DeleteHookFunc  extension.DeleteHookFunc
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

	createHook := c.CreateHookFunc(c.ObjectType)
	result, err = createHook.OnAPI(ctx, func() (types.Object, error) {
		result.SetID(UUID.String())
		currentTime := time.Now().UTC()
		result.SetCreatedAt(currentTime)
		result.SetUpdatedAt(currentTime)
		return result, nil
	})

	err = c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		return createHook.OnTransaction(ctx, storage, func() (types.Object, error) {
			if _, err = storage.Create(ctx, result); err != nil {
				return nil, util.HandleStorageError(err, string(c.ObjectType))
			}
			return result, nil
		})
	})
	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusCreated, result)
}

func (c *Controller) DeleteObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %ss...", c.ObjectType)
	deleteHook := c.DeleteHookFunc(c.ObjectType)
	if err := deleteHook.OnAPI(ctx, func(ctx context.Context) error {
		if err := c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
			return deleteHook.OnStorage(ctx, storage, func() (types.ObjectList, error) {
				return c.Repository.Delete(ctx, c.ObjectType, query.CriteriaForContext(ctx)...)
			})
		}); err != nil {
			return util.HandleSelectionError(err, string(c.ObjectType))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *Controller) DeleteSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[c.PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %s with id %s", c.ObjectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)

	deleteHook := c.DeleteHookFunc(c.ObjectType)
	err := deleteHook.OnAPI(ctx, func(ctx context.Context) error {
		if err := c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
			return deleteHook.OnStorage(ctx, storage, func() (list types.ObjectList, e error) {
				return c.Repository.Delete(ctx, c.ObjectType, byID)
			})
		}); err != nil {
			return util.HandleStorageError(err, string(c.ObjectType))
		}
		return nil
	}, byID)
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

	updateHook := c.UpdateHookFunc(c.ObjectType)

	result, err = updateHook.OnAPI(ctx, result, func(modifiedObject types.Object) (types.Object, error) {
		// overwrite object with values from API
		if err := util.BytesToObject(r.Body, modifiedObject); err != nil {
			return nil, err
		}
		// do not allow overriding these values via the API
		modifiedObject.SetID(objectID)
		modifiedObject.SetCreatedAt(createdAt)
		modifiedObject.SetUpdatedAt(time.Now().UTC())

		return modifiedObject, nil
	}, changes...)

	err = c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		return updateHook.OnTransaction(ctx, storage, func() (oldObject, newObject types.Object, err error) {
			updatedObject, err := storage.Update(ctx, result, changes...)
			return result, updatedObject, err
		})
	})
	if err != nil {
		return nil, err
	}

	result.SetCredentials(nil)
	return util.NewJSONResponse(http.StatusOK, result)
}
