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

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

const (
	reqPlatformID = "platform_id"
)

// Controller platform controller
type Controller struct {
	PlatformStorage storage.Platform
	Encrypter       security.Encrypter
}

var _ web.Controller = &Controller{}

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

	if _, err := c.PlatformStorage.Create(ctx, platform); err != nil {
		return nil, util.HandleStorageError(err, "platform", platform.ID)
	}
	platform.Credentials.Basic.Password = plainPassword
	return util.NewJSONResponse(http.StatusCreated, platform)
}

// getPlatform handler for GET /v1/platforms/:platform_id
func (c *Controller) getPlatform(r *web.Request) (*web.Response, error) {
	platformID := r.PathParams[reqPlatformID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting platform with id %s", platformID)

	platform, err := c.PlatformStorage.Get(ctx, platformID)
	if err = util.HandleStorageError(err, "platform", platformID); err != nil {
		return nil, err
	}
	platform.Credentials = nil
	return util.NewJSONResponse(http.StatusOK, platform)
}

// listPlatforms handler for GET /v1/platforms
func (c *Controller) listPlatforms(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debug("Getting all platforms")
	platforms, err := c.PlatformStorage.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, platform := range platforms {
		platform.Credentials = nil
	}

	return util.NewJSONResponse(http.StatusOK, struct {
		Platforms []*types.Platform `json:"platforms"`
	}{
		Platforms: platforms,
	})
}

// deletePlatform handler for DELETE /v1/platforms/:platform_id
func (c *Controller) deletePlatform(r *web.Request) (*web.Response, error) {
	platformID := r.PathParams[reqPlatformID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting platform with id %s", platformID)

	if err := c.PlatformStorage.Delete(ctx, platformID); err != nil {
		return nil, util.HandleStorageError(err, "platform", platformID)
	}

	// map[string]string{} will result in empty JSON
	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// updatePlatform handler for PATCH /v1/platforms/:platform_id
func (c *Controller) patchPlatform(r *web.Request) (*web.Response, error) {
	platformID := r.PathParams[reqPlatformID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating platform with id %s", platformID)

	platform, err := c.PlatformStorage.Get(ctx, platformID)
	if err != nil {
		return nil, util.HandleStorageError(err, "platform", platformID)
	}

	createdAt := platform.CreatedAt

	if err := util.BytesToObject(r.Body, platform); err != nil {
		return nil, err
	}

	platform.ID = platformID
	platform.CreatedAt = createdAt
	platform.UpdatedAt = time.Now().UTC()

	if err := c.PlatformStorage.Update(ctx, platform); err != nil {
		return nil, util.HandleStorageError(err, "platform", platformID)
	}

	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, platform)
}
