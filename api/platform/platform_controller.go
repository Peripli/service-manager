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
	"errors"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/api/common"
	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/types"
	"github.com/Peripli/service-manager/util"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const reqPlatformID = "platform_id"

// Controller platform controller
type Controller struct {
	PlatformStorage storage.Platform
}

func getPlatformFromRequest(req *filter.Request) (*types.Platform, error) {
	var platform types.Platform
	return &platform, rest.ReadJSONBody(req, &platform)
}

func checkPlatformMandatoryProperties(platform *types.Platform) error {
	if platform.Type == "" {
		return errors.New("Missing platform type")
	}
	if platform.Name == "" {
		return errors.New("Missing platform name")
	}
	return nil
}

// createPlatform handler for POST /v1/platforms
func (c *Controller) createPlatform(request *filter.Request) (*filter.Response, error) {
	logrus.Debug("Creating new platform")

	platform, errDecode := getPlatformFromRequest(request)
	if errDecode != nil {
		return nil, errDecode
	}
	if errMandatoryProperties := checkPlatformMandatoryProperties(platform); errMandatoryProperties != nil {
		return nil, types.NewErrorResponse(errMandatoryProperties, http.StatusBadRequest, "BadRequest")
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

	username, password, err := util.GenerateCredentials()
	if err != nil {
		logrus.Error("Could not generate credentials for platform")
		return nil, err
	}
	platform.Credentials = types.NewBasicCredentials(username, password)
	err = common.HandleUniqueError(c.PlatformStorage.Create(platform), "platform")
	if err != nil {
		return nil, err
	}

	return rest.NewJSONResponse(http.StatusCreated, platform)
}

// getPlatform handler for GET /v1/platforms/:platform_id
func (c *Controller) getPlatform(request *filter.Request) (*filter.Response, error) {
	platformID := request.PathParams[reqPlatformID]
	logrus.Debugf("Getting platform with id %s", platformID)

	platform, err := c.PlatformStorage.Get(platformID)
	if err = common.HandleNotFoundError(err, "platform", platformID); err != nil {
		return nil, err
	}
	return rest.NewJSONResponse(http.StatusOK, platform)
}

// getAllPlatforms handler for GET /v1/platforms
func (c *Controller) getAllPlatforms(request *filter.Request) (*filter.Response, error) {
	logrus.Debug("Getting all platforms")
	platforms, err := c.PlatformStorage.GetAll()
	if err != nil {
		return nil, err
	}
	platformsResponse := map[string][]types.Platform{"platforms": platforms}

	return rest.NewJSONResponse(http.StatusOK, &platformsResponse)
}

// deletePlatform handler for DELETE /v1/platforms/:platform_id
func (c *Controller) deletePlatform(request *filter.Request) (*filter.Response, error) {
	platformID := request.PathParams[reqPlatformID]
	logrus.Debugf("Deleting platform with id %s", platformID)

	err := c.PlatformStorage.Delete(platformID)
	if err = common.HandleNotFoundError(err, "platform", platformID); err != nil {
		return nil, err
	}
	// map[string]string{} will result in empty JSON
	return rest.NewJSONResponse(http.StatusOK, map[string]string{})
}

// updatePlatform handler for PATCH /v1/platforms/:platform_id
func (c *Controller) patchPlatform(request *filter.Request) (*filter.Response, error) {
	platformID := request.PathParams[reqPlatformID]
	logrus.Debugf("Updating platform with id %s", platformID)
	newPlatform, errDecode := getPlatformFromRequest(request)
	if errDecode != nil {
		return nil, errDecode
	}
	newPlatform.ID = platformID
	newPlatform.UpdatedAt = time.Now().UTC()
	platformStorage := c.PlatformStorage
	err := platformStorage.Update(newPlatform)
	err = common.CheckErrors(
		common.HandleNotFoundError(err, "platform", platformID),
		common.HandleUniqueError(err, "platform"),
	)
	if err != nil {
		return nil, err
	}
	platform, err := platformStorage.Get(platformID)
	if err != nil {
		return nil, err
	}
	return rest.NewJSONResponse(http.StatusOK, platform)
}
