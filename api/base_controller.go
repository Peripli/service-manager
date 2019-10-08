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

	"github.com/tidwall/sjson"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const PathParamID = "id"

// pagingLimitOffset is a constant which is needed to identify if there are more items in the DB.
// If there is 1 more item than requested, we need to generate a token for the next page.
// The last item is omitted.
const pagingLimitOffset = 1

// BaseController provides common CRUD handlers for all object types in the service manager
type BaseController struct {
	resourceBaseURL string
	objectType      types.ObjectType
	repository      storage.Repository
	objectBlueprint func() types.Object
	DefaultPageSize int
	MaxPageSize     int
}

// NewController returns a new base controller
func NewController(repository storage.Repository, resourceBaseURL string, objectType types.ObjectType, objectBlueprint func() types.Object, defaultPageSize, maxPageSize int) *BaseController {
	return &BaseController{
		repository:      repository,
		resourceBaseURL: resourceBaseURL,
		objectBlueprint: objectBlueprint,
		objectType:      objectType,
		DefaultPageSize: defaultPageSize,
		MaxPageSize:     maxPageSize,
	}
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
	result.SetCreatedAt(currentTime)
	result.SetUpdatedAt(currentTime)

	createdObj, err := c.repository.Create(ctx, result)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.objectType))
	}

	return util.NewJSONResponse(http.StatusCreated, createdObj)
}

// DeleteObjects handles the deletion of the objects specified in the request
func (c *BaseController) DeleteObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %ss...", c.objectType)

	criteria := query.CriteriaForContext(ctx)
	if _, err := c.repository.Delete(ctx, c.objectType, criteria...); err != nil {
		return nil, util.HandleStorageError(err, string(c.objectType))
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// DeleteSingleObject handles the deletion of the object with the id specified in the request
func (c *BaseController) DeleteSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting %s with id %s", c.objectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)
	ctx, err := query.AddCriteria(ctx, byID)
	if err != nil {
		return nil, err
	}
	r.Request = r.WithContext(ctx)

	return c.DeleteObjects(r)
}

// GetSingleObject handles the fetching of a single object with the id specified in the request
func (c *BaseController) GetSingleObject(r *web.Request) (*web.Response, error) {
	objectID := r.PathParams[PathParamID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting %s with id %s", c.objectType, objectID)

	byID := query.ByField(query.EqualsOperator, "id", objectID)
	var err error
	ctx, err = query.AddCriteria(ctx, byID)
	if err != nil {
		return nil, err
	}
	criteria := query.CriteriaForContext(ctx)
	object, err := c.repository.Get(ctx, c.objectType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.objectType))
	}

	stripCredentials(ctx, object)

	return util.NewJSONResponse(http.StatusOK, object)
}

// ListObjects handles the fetching of all objects
func (c *BaseController) ListObjects(r *web.Request) (*web.Response, error) {
	ctx := r.Context()

	criteria := query.CriteriaForContext(ctx)
	count, err := c.repository.Count(ctx, c.objectType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.objectType))
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
		return nil, util.HandleStorageError(err, string(c.objectType))
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
	objectID := r.PathParams[PathParamID]
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
		return nil, util.HandleStorageError(err, string(c.objectType))
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

	labels, _, _ := query.ApplyLabelChangesToLabels(labelChanges, objFromDB.GetLabels())
	objFromDB.SetLabels(labels)

	object, err := c.repository.Update(ctx, objFromDB, labelChanges)
	if err != nil {
		return nil, util.HandleStorageError(err, string(c.objectType))
	}

	stripCredentials(ctx, object)

	return util.NewJSONResponse(http.StatusOK, object)
}

func stripCredentials(ctx context.Context, object types.Object) {
	if secured, ok := object.(types.Secured); ok {
		secured.SetCredentials(nil)
	} else {
		log.C(ctx).Debugf("Object of type %s with id %s is not secured, so no credentials are cleaned up on response", object.GetType(), object.GetID())
	}
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
		stripCredentials(ctx, obj)
		page.Items = append(page.Items, obj)
	}

	if len(page.Items) > limit {
		page.Items = page.Items[:len(page.Items)-1]
		page.Token = generateTokenForItem(page.Items[len(page.Items)-1])
	}
	return page
}
