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
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/types"
	"github.com/Peripli/service-manager/util"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const reqPlatformID = "platform_id"

// Controller platform controller
type Controller struct {
	PlatformStorage storage.Platform
}

func getPlatformID(req *http.Request) string {
	return mux.Vars(req)[reqPlatformID]
}

func getPlatformFromRequest(req *http.Request) (*types.Platform, error) {
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
func (ctrl *Controller) createPlatform(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debug("Creating new platform")

	platform, errDecode := getPlatformFromRequest(req)
	if errDecode != nil {
		return errDecode
	}
	if errMandatoryProperties := checkPlatformMandatoryProperties(platform); errMandatoryProperties != nil {
		return types.NewErrorResponse(errMandatoryProperties, http.StatusBadRequest, "BadRequest")
	}
	if platform.ID == "" {
		uuid, err := uuid.NewV4()
		if err != nil {
			logrus.Error("Could not generate GUID")
			return err
		}
		platform.ID = uuid.String()
	}
	currentTime := time.Now().UTC()
	platform.CreatedAt = currentTime
	platform.UpdatedAt = currentTime

	username, password, err := util.GenerateCredentials()
	if err != nil {
		logrus.Error("Could not generate credentials for platform")
		return err
	}
	platform.Credentials = types.NewBasicCredentials(username, password)
	err = common.HandleUniqueError(ctrl.PlatformStorage.Create(platform), "platform")
	if err != nil {
		return err
	}

	return rest.SendJSON(rw, http.StatusCreated, platform)
}

// getPlatform handler for GET /v1/platforms/:platform_id
func (ctrl *Controller) getPlatform(rw http.ResponseWriter, req *http.Request) error {
	platformID := getPlatformID(req)
	logrus.Debugf("Getting platform with id %s", platformID)

	platform, err := ctrl.PlatformStorage.Get(platformID)
	if err = common.HandleNotFoundError(err, "platform", platformID); err != nil {
		return err
	}
	return rest.SendJSON(rw, http.StatusOK, platform)
}

// getAllPlatforms handler for GET /v1/platforms
func (ctrl *Controller) getAllPlatforms(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debug("Getting all platforms")
	platforms, err := ctrl.PlatformStorage.GetAll()
	if err != nil {
		return err
	}
	platformsResponse := map[string][]types.Platform{"platforms": platforms}

	return rest.SendJSON(rw, http.StatusOK, &platformsResponse)
}

// deletePlatform handler for DELETE /v1/platforms/:platform_id
func (ctrl *Controller) deletePlatform(rw http.ResponseWriter, req *http.Request) error {
	platformID := getPlatformID(req)
	logrus.Debugf("Deleting platform with id %s", platformID)

	err := ctrl.PlatformStorage.Delete(platformID)
	if err = common.HandleNotFoundError(err, "platform", platformID); err != nil {
		return err
	}
	// map[string]string{} will result in empty JSON
	return rest.SendJSON(rw, http.StatusOK, map[string]string{})
}

// updatePlatform handler for PATCH /v1/platforms/:platform_id
func (ctrl *Controller) updatePlatform(rw http.ResponseWriter, req *http.Request) error {
	platformID := getPlatformID(req)
	logrus.Debugf("Updating platform with id %s", platformID)
	newPlatform, errDecode := getPlatformFromRequest(req)
	if errDecode != nil {
		return errDecode
	}
	newPlatform.ID = platformID
	newPlatform.UpdatedAt = time.Now().UTC()
	platformStorage := ctrl.PlatformStorage
	err := platformStorage.Update(newPlatform)
	err = common.CheckErrors(
		common.HandleNotFoundError(err, "platform", platformID),
		common.HandleUniqueError(err, "platform"),
	)
	if err != nil {
		return err
	}
	platform, err := platformStorage.Get(platformID)
	if err != nil {
		return err
	}
	return rest.SendJSON(rw, http.StatusOK, platform)
}
