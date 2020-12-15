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

	supportsCascadeDelete bool
}

// NewController returns a new base controller
func NewController(ctx context.Context, options *Options, resourceBaseURL string, objectType types.ObjectType, objectBlueprint func() types.Object, supportsCascadeDelete bool) *BaseController {
	poolSize := options.OperationSettings.DefaultPoolSize
	for _, pool := range options.OperationSettings.Pools {
		if pool.Resource == objectType.String() {
			poolSize = pool.Size
			break
		}
	}
	controller := &BaseController{
		repository:            options.Repository,
		resourceBaseURL:       resourceBaseURL,
		objectBlueprint:       objectBlueprint,
		objectType:            objectType,
		DefaultPageSize:       options.APISettings.DefaultPageSize,
		MaxPageSize:           options.APISettings.MaxPageSize,
		scheduler:             operations.NewScheduler(ctx, options.Repository, options.OperationSettings, poolSize, options.WaitGroup),
		supportsCascadeDelete: supportsCascadeDelete,
	}

	return controller
}

// NewAsyncController returns a new base controller with a scheduler making it effectively an async controller
func NewAsyncController(ctx context.Context, options *Options, resourceBaseURL string, objectType types.ObjectType, isAsyncDefault bool, objectBlueprint func() types.Object, supportsCascadeDelete bool) *BaseController {
	controller := NewController(ctx, options, resourceBaseURL, objectType, objectBlueprint, supportsCascadeDelete)
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
				Path:   fmt.Sprintf("%s/{%s}%s/{%s}", c.resourceBaseURL, web.PathParamResourceID, web.ResourceOperationsURL, web.PathParamID),
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
	if err := util.ValidateJSONContentType(r.Header.Get("Content-Type")); err != nil {
		return nil, err
	}

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
		Context:       c.prepareOperationContextByRequest(r),
	}

	createdObj, isAsync, err := c.scheduler.ScheduleStorageAction(ctx, operation, action, c.supportsAsync)
	if err != nil {
		return nil, err
	}

	if isAsync {
		return util.NewLocationResponse(operation.GetID(), operation.ResourceID, c.resourceBaseURL)
	}

	if err := attachLastOperation(ctx, createdObj.GetID(), createdObj, c.repository); err != nil {
		return nil, err
	}

	cleanObject(ctx, createdObj.GetLastOperation())
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
	resourceID := r.PathParams[web.PathParamResourceID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %s with id %s", c.objectType, resourceID)

	byID := query.ByField(query.EqualsOperator, "id", resourceID)
	ctx, err := query.AddCriteria(ctx, byID)
	if err != nil {
		return nil, err
	}
	r.Request = r.WithContext(ctx)
	criteria := query.CriteriaForContext(ctx)
	opCtx := c.prepareOperationContextByRequest(r)

	action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		// At this point, the resource will be already deleted if cascade operation requested.
		if c.supportsCascadeDelete && opCtx.Cascade {
			return nil, nil
		}
		err := repository.Delete(ctx, c.objectType, criteria...)
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", c.objectType, err)
	}
	var cascadeRootId = ""
	if opCtx.Cascade {
		if c.supportsCascadeDelete {
			// Scan if requested resource really exists
			resources, err := c.repository.List(ctx, c.objectType, criteria...)
			if err != nil {
				return nil, util.HandleStorageError(err, c.objectType.String())
			}
			if resources.Len() == 0 {
				return nil, &util.HTTPError{
					ErrorType:   "NotFound",
					Description: "Resource not found",
					StatusCode:  http.StatusNotFound,
				}
			}
			cascadeRootId = UUID.String()
		} else {
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: "Cascade delete is not supported for this API",
				StatusCode:  http.StatusBadRequest,
			}
		}
	}
	if c.supportsCascadeDelete && opCtx.Cascade {
		concurrentOp, err := operations.FindCascadeOperationForResource(ctx, c.repository, resourceID)
		if err != nil {
			return nil, err
		}
		if concurrentOp != nil {
			return util.NewLocationResponse(concurrentOp.GetID(), resourceID, c.resourceBaseURL)
		}
	}

	isForce := r.URL.Query().Get(web.QueryParamForce) == "true"
	labels := types.Labels{}
	if isForce {
		labels["force"] = []string{"true"}
	}

	operation := &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Labels:    labels,
			Ready:     true,
		},
		Type:          types.DELETE,
		State:         types.IN_PROGRESS,
		ResourceID:    resourceID,
		ResourceType:  c.objectType,
		PlatformID:    types.SMPlatform,
		CorrelationID: log.CorrelationIDFromContext(ctx),
		Context:       opCtx,
		CascadeRootID: cascadeRootId,
	}
	if c.supportsCascadeDelete && opCtx.Cascade {
		_, err = c.scheduler.ScheduleSyncStorageAction(ctx, operation, action)
		if err != nil {
			return nil, err
		}
		return util.NewLocationResponse(operation.GetID(), operation.ResourceID, c.resourceBaseURL)
	}
	_, isAsync, err := c.scheduler.ScheduleStorageAction(ctx, operation, action, c.supportsAsync)
	if err != nil {
		return nil, err
	}

	if isAsync {
		return util.NewLocationResponse(operation.GetID(), operation.ResourceID, c.resourceBaseURL)
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

	if err := attachLastOperation(ctx, objectID, object, c.repository); err != nil {
		return nil, err
	}

	cleanObject(ctx, object.GetLastOperation())
	return util.NewJSONResponse(http.StatusOK, object)
}

// GetOperation handles the fetching of a single operation with the id specified for the specified resource
func (c *BaseController) GetOperation(r *web.Request) (*web.Response, error) {
	return GetResourceOperation(r, c.repository, c.objectType)
}

func GetResourceOperation(r *web.Request, repository storage.Repository, objectType types.ObjectType) (*web.Response, error) {
	objectID := r.PathParams[web.PathParamResourceID]
	operationID := r.PathParams[web.PathParamID]

	ctx := r.Context()
	log.C(ctx).Debugf("Getting operation with id %s for object of type %s with id %s", operationID, objectType, objectID)

	byOperationID := query.ByField(query.EqualsOperator, "id", operationID)
	byObjectID := query.ByField(query.EqualsOperator, "resource_id", objectID)
	var err error
	ctx, err = query.AddCriteria(ctx, byObjectID, byOperationID)
	if err != nil {
		return nil, err
	}
	criteria := query.CriteriaForContext(ctx)
	operation, err := repository.Get(ctx, types.OperationType, criteria...)
	cleanObject(ctx, operation)
	if err != nil {
		return nil, util.HandleStorageError(err, objectType.String())
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

	attachLastOps := r.URL.Query().Get("attach_last_operations")
	if attachLastOps == "true" {
		if err := attachLastOperations(ctx, objectList, c.repository); err != nil {
			return nil, err
		}
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
	if err := util.ValidateJSONContentType(r.Header.Get("Content-Type")); err != nil {
		return nil, err
	}

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
		Context:       c.prepareOperationContextByRequest(r),
	}

	object, isAsync, err := c.scheduler.ScheduleStorageAction(ctx, operation, action, c.supportsAsync)
	if err != nil {
		return nil, err
	}

	if isAsync {
		return util.NewLocationResponse(operation.GetID(), operation.ResourceID, c.resourceBaseURL)
	}

	if err := attachLastOperation(ctx, object.GetID(), object, c.repository); err != nil {
		return nil, err
	}

	cleanObject(ctx, object.GetLastOperation())
	cleanObject(ctx, object)
	return util.NewJSONResponse(http.StatusOK, object)
}

func cleanObject(ctx context.Context, object types.Object) {
	if secured, ok := object.(types.Strip); ok {
		secured.Sanitize(ctx)
	}
}
func getResourceIds(resources types.ObjectList) []string {
	var resourceIds []string
	for i := 0; i < resources.Len(); i++ {
		resource := resources.ItemAt(i)
		resourceIds = append(resourceIds, resource.GetID())
	}
	return resourceIds
}

func attachLastOperations(ctx context.Context, resources types.ObjectList, repository storage.Repository) error {
	lastOperationsMap, err := getLastOperations(ctx, getResourceIds(resources), repository)

	if err != nil {
		return err
	}

	for i := 0; i < resources.Len(); i++ {
		resource := resources.ItemAt(i)
		if LastOp, ok := lastOperationsMap[resource.GetID()]; ok {
			LastOp.TransitiveResources = nil
			resource.SetLastOperation(LastOp)
		}
	}

	return nil
}

func getLastOperations(ctx context.Context, resourceIDs []string, repository storage.Repository) (map[string]*types.Operation, error) {
	if len(resourceIDs) == 0 {
		return nil, nil
	}

	queryParams := map[string]interface{}{
		"id_list": resourceIDs,
	}

	resourceLastOps, err := repository.QueryForList(
		ctx,
		types.OperationType,
		storage.QueryForLastOperationsPerResource,
		queryParams)

	if err != nil {
		return nil, util.HandleStorageError(err, types.OperationType.String())
	}

	instanceLastOpsMap := make(map[string]*types.Operation)

	for i := 0; i < resourceLastOps.Len(); i++ {
		lastOp := resourceLastOps.ItemAt(i).(*types.Operation)
		instanceLastOpsMap[lastOp.ResourceID] = lastOp
	}

	return instanceLastOpsMap, nil
}

func attachLastOperation(ctx context.Context, objectID string, object types.Object, repository storage.Repository) error {
	ops, err := getLastOperations(ctx, []string{objectID}, repository)

	if err != nil {
		return err
	}

	if lastOperation, ok := ops[objectID]; ok {
		lastOperation.TransitiveResources = nil
		object.SetLastOperation(lastOperation)
	}

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

func (c *BaseController) prepareOperationContextByRequest(r *web.Request) *types.OperationContext {
	operationContext := &types.OperationContext{}
	async := r.URL.Query().Get(web.QueryParamAsync)
	cascade := r.URL.Query().Get(web.QueryParamCascade)

	if async == "" {
		operationContext.Async = false
		operationContext.IsAsyncNotDefined = true
	} else if async == "false" {
		operationContext.Async = false
		operationContext.IsAsyncNotDefined = false
	} else {
		operationContext.Async = true
		operationContext.IsAsyncNotDefined = false
	}

	if cascade == "true" {
		operationContext.Cascade = true
		if operationContext.IsAsyncNotDefined {
			operationContext.Async = true
		}
	} else {
		operationContext.Cascade = false
	}

	return operationContext
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
