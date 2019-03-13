/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package platform

import (
	"net/http"
	"time"

	"github.com/Peripli/service-manager/api/base"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

// Controller platform controller
type Controller struct {
	*base.Controller
}

var _ web.Controller = &Controller{}

func NewController(repository storage.Repository, encrypter security.Encrypter) *Controller {
	baseController := base.NewController(repository, web.PlatformsURL, func() types.Object {
		return &types.Platform{}
	})

	return &Controller{
		Controller: baseController,
	}
}

// createPlatform handler for POST /v1/platforms
func (c *Controller) createPlatform(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logger := log.C(ctx)
	logger.Debug("Creating new platform")

	platform := &types.Platform{}
	if err := util.BytesToObject(r.Body, platform); err != nil {
		return nil, err
	}

	if platform.ID == "" {
		UUID, err := uuid.NewV4()
		if err != nil {
			logger.Error("Could not generate GUID")
			return nil, err
		}
		platform.ID = UUID.String()
	}
	currentTime := time.Now().UTC()
	platform.CreatedAt = currentTime
	platform.UpdatedAt = currentTime

	credentials, err := types.GenerateCredentials()
	if err != nil {
		logger.Error("Could not generate credentials for platform")
		return nil, err
	}
	plainPassword := credentials.Basic.Password
	transformedPassword, err := c.Encrypter.Encrypt(ctx, []byte(plainPassword))
	if err != nil {
		return nil, err
	}
	credentials.Basic.Password = string(transformedPassword)
	platform.Credentials = credentials

	if _, err := c.Repository.Create(ctx, platform); err != nil {
		return nil, util.HandleStorageError(err, "platform")
	}
	platform.Credentials.Basic.Password = plainPassword
	return web.NewJSONResponse(http.StatusCreated, platform)
}

// getPlatform handler for GET /v1/platforms/:platform_id
func (c *Controller) getPlatform(r *web.Request) (*web.Response, error) {
	platformID := r.PathParams[reqPlatformID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting platform with id %s", platformID)

	platform, err := c.Repository.Get(ctx, platformID, types.PlatformType)
	if err = util.HandleStorageError(err, "platform"); err != nil {
		return nil, err
	}
	platform.(*types.Platform).Credentials = nil
	return web.NewJSONResponse(http.StatusOK, platform)
}

// listPlatforms handler for GET /v1/platforms
func (c *Controller) listPlatforms(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debug("Getting all platforms")
	platforms, err := c.Repository.List(ctx, types.PlatformType, query.CriteriaForContext(ctx)...)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}

	for i := 0; i < platforms.Len(); i++ {
		platform := platforms.ItemAt(i)
		platform.(*types.Platform).Credentials = nil
	}

	return web.NewJSONResponse(http.StatusOK, platforms)
}

func (c *Controller) deletePlatforms(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting visibilities...")

	if _, err := c.Repository.Delete(ctx, types.PlatformType, query.CriteriaForContext(ctx)...); err != nil {
		return nil, util.HandleSelectionError(err, "platform")
	}
	return web.NewJSONResponse(http.StatusOK, map[string]string{})
}

// deletePlatform handler for DELETE /v1/platforms/:platform_id
func (c *Controller) deletePlatform(r *web.Request) (*web.Response, error) {
	platformID := r.PathParams[reqPlatformID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting platform with id %s", platformID)

	byIDQuery := query.ByField(query.EqualsOperator, "id", platformID)
	if _, err := c.Repository.Delete(ctx, types.PlatformType, byIDQuery); err != nil {
		return nil, util.HandleStorageError(err, "platform")
	}

	// map[string]string{} will result in empty JSON
	return web.NewJSONResponse(http.StatusOK, map[string]string{})
}

// updatePlatform handler for PATCH /v1/platforms/:platform_id
func (c *Controller) patchPlatform(r *web.Request) (*web.Response, error) {
	platformID := r.PathParams[reqPlatformID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating platform with id %s", platformID)

	platform, err := c.Repository.Get(ctx, platformID, types.PlatformType)
	if err != nil {
		return nil, util.HandleStorageError(err, "platform")
	}

	if err := util.BytesToObject(r.Body, platform); err != nil {
		return nil, err
	}

	// TODO: defaulting
	if platform, err = c.Repository.Update(ctx, platform); err != nil {
		return nil, util.HandleStorageError(err, "platform")
	}

	if err != nil {
		return nil, err
	}

	platform.Credentials = nil
	return web.NewJSONResponse(http.StatusOK, platform)
}
