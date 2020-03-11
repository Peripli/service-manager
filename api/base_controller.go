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

package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Peripli/service-manager/operations"

	"github.com/tidwall/sjson"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

// pagingLimitOffset is a constant which is needed to identify if there are more items in the DB.
// If there is 1 more item than requested, we need to generate a token for the next page.
// The last item is omitted.
const pagingLimitOffset = 1

// BaseController provides common CRUD handlers for all object types in the service manager
type BaseController struct {
	scheduler *operations.Scheduler

	resourceBaseURL string
	objectType      types.ObjectType
	repository      storage.Repository
	objectBlueprint func() types.Object

	DefaultPageSize int
	MaxPageSize     int

	supportsAsync  bool
	isAsyncDefault bool
}

// NewController returns a new base controller
func NewController(ctx context.Context, options *Options, resourceBaseURL string, objectType types.ObjectType, objectBlueprint func() types.Object) *BaseController {
	poolSize := options.OperationSettings.DefaultPoolSize
	for _, pool := range options.OperationSettings.Pools {
		if pool.Resource == objectType.String() {
			poolSize = pool.Size
			break
		}
	}
	controller := &BaseController{
		repository:      options.Repository,
		resourceBaseURL: resourceBaseURL,
		objectBlueprint: objectBlueprint,
		objectType:      objectType,
		DefaultPageSize: options.APISettings.DefaultPageSize,
		MaxPageSize:     options.APISettings.MaxPageSize,
		scheduler:       operations.NewScheduler(ctx, options.Repository, options.OperationSettings, poolSize, options.WaitGroup),
	}

	return controller
}

// NewAsyncController returns a new base controller with a scheduler making it effectively an async controller
func NewAsyncController(ctx context.Context, options *Options, resourceBaseURL string, objectType types.ObjectType, isAsyncDefault bool, objectBlueprint func() types.Object) *BaseController {
	controller := NewController(ctx, options, resourceBaseURL, objectType, objectBlueprint)
	controller.supportsAsync = true
	controller.isAsyncDefault = isAsyncDefault

	return controller
}

// Routes returns the common set of routes for all objects
func (c *BaseController) Routes() []web.Route {
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
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.GetSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}%s/{%s}", c.resourceBaseURL, web.PathParamResourceID, web.OperationsURL, web.PathParamID),
			},
			Handler: c.GetOperation,
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
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.DeleteSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.PatchObject,
		},
	}
}

// CreateObject handles the creation of a new object
func (c *BaseController) CreateObject(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Creating new %s", c.objectType)

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
	// override ready provide from the request body
	result.SetCreatedAt(currentTime)
	result.SetUpdatedAt(currentTime)
	result.SetReady(false)

	action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		object, err := repository.Create(ctx, result)
		return object, util.HandleStorageError(err, c.objectType.String())
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", c.objectType, err)
	}
	operation := &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Type:          types.CREATE,
		State:         types.IN_PROGRESS,
		ResourceID:    result.GetID(),
		ResourceType:  c.objectType,
		PlatformID:    types.SMPlatform,
		CorrelationID: log.CorrelationIDFromContext(ctx),
	}

	if c.shouldExecuteAsync(r) {
		log.C(ctx).Debugf("Request will be executed asynchronously")
		if err := c.checkAsyncSupport(); err != nil {
			return nil, err
		}

		if err := c.scheduler.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
			return nil, err
		}

		return newAsyncResponse(operation.GetID(), result.GetID(), c.resourceBaseURL)
	}

	log.C(ctx).Debugf("Request will be executed synchronously")
	createdObj, err := c.scheduler.ScheduleSyncStorageAction(ctx, operation, action)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusCreated, createdObj)
}

// DeleteObjects handles the deletion of the objects specified in the request
func (c *BaseController) DeleteObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %ss...", c.objectType)

	isAsync := r.URL.Query().Get(web.QueryParamAsync)
	if isAsync == "true" {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Only one resource can be deleted asynchronously at a time",
			StatusCode:  http.StatusBadRequest,
		}
	}

	criteria := query.CriteriaForContext(ctx)

	log.C(ctx).Debugf("Request will be executed synchronously")
	if err := c.repository.Delete(ctx, c.objectType, criteria...); err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// DeleteSingleObject handles the deletion of the object with the id specified in the request
func (c *BaseController) DeleteSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[web.PathParamResourceID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %s with id %s", c.objectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)
	ctx, err := query.AddCriteria(ctx, byID)
	if err != nil {
		return nil, err
	}
	r.Request = r.WithContext(ctx)
	criteria := query.CriteriaForContext(ctx)

	action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		err := repository.Delete(ctx, c.objectType, criteria...)
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", c.objectType, err)
	}
	operation := &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Type:          types.DELETE,
		State:         types.IN_PROGRESS,
		ResourceID:    objectID,
		ResourceType:  c.objectType,
		PlatformID:    types.SMPlatform,
		CorrelationID: log.CorrelationIDFromContext(ctx),
	}

	if c.shouldExecuteAsync(r) {
		log.C(ctx).Debugf("Request will be executed asynchronously")
		if err := c.checkAsyncSupport(); err != nil {
			return nil, err
		}

		if err := c.scheduler.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
			return nil, err
		}

		return newAsyncResponse(operation.ID, objectID, c.resourceBaseURL)
	}

	log.C(ctx).Debugf("Request will be executed synchronously")
	if _, err := c.scheduler.ScheduleSyncStorageAction(ctx, operation, action); err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// GetSingleObject handles the fetching of a single object with the id specified in the request
func (c *BaseController) GetSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[web.PathParamResourceID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting %s with id %s", c.objectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)
	criteria := query.CriteriaForContext(ctx)
	object, err := c.repository.Get(ctx, c.objectType, append(criteria, byID)...)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	cleanObject(ctx, object)

	if err := attachLastOperation(ctx, objectID, object, r, c.repository); err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, object)
}

// GetOperation handles the fetching of a single operation with the id specified for the specified resource
func (c *BaseController) GetOperation(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[web.PathParamResourceID]
	operationID := r.PathParams[web.PathParamID]

	ctx := r.Context()
	log.C(ctx).Debugf("Getting operation with id %s for object of type %s with id %s", operationID, c.objectType, objectID)

	byOperationID := query.ByField(query.EqualsOperator, "id", operationID)
	byObjectID := query.ByField(query.EqualsOperator, "resource_id", objectID)
	var err error
	ctx, err = query.AddCriteria(ctx, byObjectID, byOperationID)
	if err != nil {
		return nil, err
	}
	criteria := query.CriteriaForContext(ctx)
	operation, err := c.repository.Get(ctx, types.OperationType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusOK, operation)
}

// ListObjects handles the fetching of all objects
func (c *BaseController) ListObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()

	criteria := query.CriteriaForContext(ctx)
	count, err := c.repository.Count(ctx, c.objectType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	maxItems := r.URL.Query().Get("max_items")
	limit, err := c.parseMaxItemsQuery(maxItems)
	if err != nil {
		return nil, err
	}

	if limit == 0 {
		log.C(ctx).Debugf("Returning only count of %s since max_items is 0", c.objectType)
		page := struct {
			ItemsCount int `json:"num_items"`
		}{
			ItemsCount: count,
		}
		return util.NewJSONResponse(http.StatusOK, page)
	}

	rawToken := r.URL.Query().Get("token")
	pagingSequence, err := c.parsePageToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	criteria = append(criteria, query.LimitResultBy(limit+pagingLimitOffset),
		query.OrderResultBy("paging_sequence", query.AscOrder),
		query.ByField(query.GreaterThanOperator, "paging_sequence", pagingSequence))

	log.C(ctx).Debugf("Getting a page of %ss", c.objectType)
	objectList, err := c.repository.List(ctx, c.objectType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	page := pageFromObjectList(ctx, objectList, count, limit)
	resp, err := util.NewJSONResponse(http.StatusOK, page)
	if err != nil {
		return nil, err
	}

	if page.Token != "" {
		nextPageUrl := r.URL
		q := nextPageUrl.Query()
		q.Set("token", page.Token)
		nextPageUrl.RawQuery = q.Encode()
		resp.Header.Add("Link", fmt.Sprintf(`<%s>; rel="next"`, nextPageUrl))
	}

	return resp, nil
}

// PatchObject handles the update of the object with the id specified in the request
func (c *BaseController) PatchObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[web.PathParamResourceID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating %s with id %s", c.objectType, objectID)

	labelChanges, err := query.LabelChangesFromJSON(r.Body)
	if err != nil {
		return nil, err
	}

	byID := query.ByField(query.EqualsOperator, "id", objectID)
	ctx, err = query.AddCriteria(ctx, byID)
	if err != nil {
		return nil, err
	}
	criteria := query.CriteriaForContext(ctx)
	objFromDB, err := c.repository.Get(ctx, c.objectType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	if r.Body, err = sjson.DeleteBytes(r.Body, "labels"); err != nil {
		return nil, err
	}
	createdAt := objFromDB.GetCreatedAt()
	updatedAt := objFromDB.GetUpdatedAt()

	if err := util.BytesToObject(r.Body, objFromDB); err != nil {
		return nil, err
	}

	objFromDB.SetID(objectID)
	objFromDB.SetCreatedAt(createdAt)
	objFromDB.SetUpdatedAt(updatedAt)
	objFromDB.SetReady(true)

	labels, _, _ := query.ApplyLabelChangesToLabels(labelChanges, objFromDB.GetLabels())
	objFromDB.SetLabels(labels)

	action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		object, err := repository.Update(ctx, objFromDB, labelChanges, criteria...)
		return object, util.HandleStorageError(err, c.objectType.String())
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", c.objectType, err)
	}
	operation := &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Type:          types.UPDATE,
		State:         types.IN_PROGRESS,
		ResourceID:    objFromDB.GetID(),
		ResourceType:  c.objectType,
		PlatformID:    types.SMPlatform,
		CorrelationID: log.CorrelationIDFromContext(ctx),
	}

	if c.shouldExecuteAsync(r) {
		log.C(ctx).Debugf("Request will be executed asynchronously")
		if err := c.checkAsyncSupport(); err != nil {
			return nil, err
		}

		if err := c.scheduler.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
			return nil, err
		}

		return newAsyncResponse(operation.GetID(), objFromDB.GetID(), c.resourceBaseURL)
	}

	log.C(ctx).Debugf("Request will be executed synchronously")
	object, err := c.scheduler.ScheduleSyncStorageAction(ctx, operation, action)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	cleanObject(ctx, object)
	return util.NewJSONResponse(http.StatusOK, object)
}

func cleanObject(ctx context.Context, object types.Object) {
	if secured, ok := object.(types.Strip); ok {
		secured.Sanitize()
	} else {
		log.C(ctx).Debugf("Object of type %s with id %s is not secured, so no credentials are cleaned up on response", object.GetType(), object.GetID())
	}
}

func attachLastOperation(ctx context.Context, objectID string, object types.Object, r *web.Request, repository storage.Repository) error {
	orderBy := query.OrderResultBy("paging_sequence", query.DescOrder)
	byObjectID := query.ByField(query.EqualsOperator, "resource_id", objectID)
	// Limit cannot be applied, otherwise the query is corrupted and does not return valid result
	list, err := repository.List(ctx, types.OperationType, byObjectID, orderBy)
	if err != nil {
		return util.HandleStorageError(err, types.OperationType.String())
	}
	if list.Len() == 0 {
		log.C(ctx).Debugf("No last operation found for entity with id %s of type %s", objectID, object.GetType().String())
		return nil
	}
	lastOperation := list.ItemAt(0)
	object.SetLastOperation(lastOperation.(*types.Operation))
	return nil
}

func (c *BaseController) parseMaxItemsQuery(maxItems string) (int, error) {
	limit := c.DefaultPageSize
	var err error
	if maxItems != "" {
		limit, err = strconv.Atoi(maxItems)
		if err != nil {
			return -1, &util.HTTPError{
				ErrorType:   "InvalidMaxItems",
				Description: fmt.Sprintf("max_items should be integer: %v", err),
				StatusCode:  http.StatusBadRequest,
			}
		}
		if limit < 0 {
			return -1, &util.HTTPError{
				ErrorType:   "InvalidMaxItems",
				Description: fmt.Sprintf("max_items cannot be negative"),
				StatusCode:  http.StatusBadRequest,
			}
		}
		if limit > c.MaxPageSize {
			limit = c.MaxPageSize
		}
	}
	return limit, nil
}

func (c *BaseController) parsePageToken(ctx context.Context, token string) (string, error) {
	targetPageSequence := "0"
	if token != "" {
		base64DecodedTokenBytes, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			log.C(ctx).Infof("Invalid token provided: %v", err)
			return "", &util.HTTPError{
				ErrorType:   "TokenInvalid",
				Description: "Invalid token provided.",
				StatusCode:  http.StatusBadRequest,
			}
		}
		targetPageSequence = string(base64DecodedTokenBytes)
		pagingSequence, err := strconv.ParseInt(targetPageSequence, 10, 0)
		if err != nil {
			log.C(ctx).Infof("Invalid token provided: %v", err)
			return "", &util.HTTPError{
				ErrorType:   "TokenInvalid",
				Description: "Invalid token provided.",
				StatusCode:  http.StatusBadRequest,
			}
		}
		if pagingSequence < 0 {
			log.C(ctx).Infof("Invalid token provided: negative value")
			return "", &util.HTTPError{
				ErrorType:   "TokenInvalid",
				Description: "Invalid token provided.",
				StatusCode:  http.StatusBadRequest,
			}
		}
	}
	return targetPageSequence, nil
}

func (c *BaseController) shouldExecuteAsync(r *web.Request) bool {
	async := r.URL.Query().Get(web.QueryParamAsync)
	if async == "" {
		return c.isAsyncDefault
	}

	return async == "true"
}

func (c *BaseController) checkAsyncSupport() error {
	if !c.supportsAsync {
		return &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("requested %s api doesn't support asynchronous operations", c.objectType),
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}

func generateTokenForItem(obj types.Object) string {
	nextPageToken := obj.GetPagingSequence()
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(nextPageToken, 10)))
}

func pageFromObjectList(ctx context.Context, objectList types.ObjectList, count, limit int) *types.ObjectPage {
	page := &types.ObjectPage{
		ItemsCount: count,
		Items:      make([]types.Object, 0, objectList.Len()),
	}

	for i := 0; i < objectList.Len(); i++ {
		obj := objectList.ItemAt(i)
		cleanObject(ctx, obj)
		page.Items = append(page.Items, obj)
	}

	if len(page.Items) > limit {
		page.Items = page.Items[:len(page.Items)-1]
		page.Token = generateTokenForItem(page.Items[len(page.Items)-1])
	}
	return page
}

func newAsyncResponse(operationID, resourceID, resourceBaseURL string) (*web.Response, error) {
	operationURL := buildOperationURL(operationID, resourceID, resourceBaseURL)
	additionalHeaders := map[string]string{"Location": operationURL}
	return util.NewJSONResponseWithHeaders(http.StatusAccepted, map[string]string{}, additionalHeaders)
}

func buildOperationURL(operationID, resourceID, resourceType string) string {
	return fmt.Sprintf("%s/%s%s/%s", resourceType, resourceID, web.OperationsURL, operationID)
}
