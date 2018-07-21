/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version oidc_authn.0 (the "License");
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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const reqPlatformID = "platform_id"

// Controller platform controller
type Controller struct {
	PlatformStorage storage.Platform
}

var _ web.Controller = &Controller{}

// createPlatform handler for POST /v1/platforms
func (c *Controller) createPlatform(request *web.Request) (*web.Response, error) {
	logrus.Debug("Creating new platform")
	var platform *types.Platform

	if err := util.UnmarshalAndValidate(request.Body, platform); err != nil {
		return nil, err
	}

	if platform.ID == "" {
		uuid, err := uuid.NewV4()
		if err != nil {
			logrus.Error("Could not generate GUID")
			return nil, err
		}
		platform.ID = uuid.String()
	}
	currentTime := time.Now().UTC()
	platform.CreatedAt = currentTime
	platform.UpdatedAt = currentTime

	credentials, err := types.GenerateCredentials()
	if err != nil {
		logrus.Error("Could not generate credentials for platform")
		return nil, err
	}
	platform.Credentials = credentials

	err = storage.HandleUniqueError(c.PlatformStorage.Create(platform), "platform")
	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusCreated, platform)
}

// getPlatform handler for GET /v1/platforms/:platform_id
func (c *Controller) getPlatform(request *web.Request) (*web.Response, error) {
	platformID := request.PathParams[reqPlatformID]
	logrus.Debugf("Getting platform with id %s", platformID)

	platform, err := c.PlatformStorage.Get(platformID)
	if err = storage.HandleNotFoundError(err, "platform", platformID); err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, platform)
}

// getAllPlatforms handler for GET /v1/platforms
func (c *Controller) getAllPlatforms(request *web.Request) (*web.Response, error) {
	logrus.Debug("Getting all platforms")
	platforms, err := c.PlatformStorage.GetAll()
	if err != nil {
		return nil, err
	}
	platformsResponse := map[string][]types.Platform{"platforms": platforms}

	return util.NewJSONResponse(http.StatusOK, &platformsResponse)
}

// deletePlatform handler for DELETE /v1/platforms/:platform_id
func (c *Controller) deletePlatform(request *web.Request) (*web.Response, error) {
	platformID := request.PathParams[reqPlatformID]
	logrus.Debugf("Deleting platform with id %s", platformID)

	err := c.PlatformStorage.Delete(platformID)
	if err = storage.HandleNotFoundError(err, "platform", platformID); err != nil {
		return nil, err
	}
	// map[string]string{} will result in empty JSON
	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// updatePlatform handler for PATCH /v1/platforms/:platform_id
func (c *Controller) patchPlatform(request *web.Request) (*web.Response, error) {
	platformID := request.PathParams[reqPlatformID]
	logrus.Debugf("Updating platform with id %s", platformID)

	platform, err := c.PlatformStorage.Get(platformID)
	err = storage.HandleNotFoundError(err, "platform", platformID)
	if err != nil {
		return nil, err
	}

	if err := util.UnmarshalAndValidate(request.Body, platform); err != nil {
		return nil, err
	}

	platform.UpdatedAt = time.Now().UTC()

	err = c.PlatformStorage.Update(platform)
	err = storage.CheckErrors(
		storage.HandleNotFoundError(err, "platform", platformID),
		storage.HandleUniqueError(err, "platform"),
	)
	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, platform)
}
