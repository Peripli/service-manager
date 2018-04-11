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
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/util"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// Controller platform controller
type Controller struct{}

func getPlatformID(req *http.Request) string {
	vars := mux.Vars(req)
	return vars["platform_id"]
}

func getPlatformFromRequest(req *http.Request) (*rest.Platform, error) {
	decoder := json.NewDecoder(req.Body)
	var platform rest.Platform
	if err := decoder.Decode(&platform); err != nil {
		return nil, err
	}
	return &platform, nil
}

func mergePlatforms(source *rest.Platform, target *rest.Platform) {
	if source.Name != "" {
		target.Name = source.Name
	}
	if source.Description != "" {
		target.Description = source.Description
	}
	if source.Type != "" {
		target.Type = source.Type
	}
	target.UpdatedAt = time.Now().UTC()
}

func checkPlatformMandatoryProperties(platform *rest.Platform) error {
	if platform.Type == "" {
		return errors.New("missing platform type")
	}
	if platform.Name == "" {
		return errors.New("missing platform name")
	}
	return nil
}

// addPlatform handler for POST /v1/platforms
func (platformCtrl *Controller) addPlatform(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("POST to %s", req.RequestURI)

	platform, errDecode := getPlatformFromRequest(req)
	if errDecode != nil {
		return registerPlatformError(errorRequestBodyDecode(errDecode), http.StatusBadRequest)
	}
	if errMandatoryProperties := checkPlatformMandatoryProperties(platform); errMandatoryProperties != nil {
		logrus.Debug(errMandatoryProperties)
		return registerPlatformError(errMandatoryProperties, http.StatusBadRequest)
	}
	username, password := util.GenerateCredentials()
	if platform.ID == "" {
		// TODO: Use uuid package
		platform.ID = util.GenerateID()
	}
	currentTime := time.Now().UTC()
	platform.CreatedAt = currentTime
	platform.UpdatedAt = currentTime

	platform.Credentials = &rest.Credentials{
		Basic: &rest.Basic{
			Username: username,
			Password: password,
		},
	}
	platformStorage := storage.Get().Platform()
	errSave := platformStorage.Create(platform)
	if errSave == storage.ConflictEntityError {
		return registerPlatformError(errSave, http.StatusConflict)
	} else if errSave != nil {
		return registerPlatformError(errorSavePlatform(errSave), http.StatusInternalServerError)
	}

	if errJSON := rest.SendJSON(rw, http.StatusCreated, platform); errJSON != nil {
		return registerPlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
	}
	return nil
}

// getPlatform handler for GET /v1/platforms/:platform_id
func (platformCtrl *Controller) getPlatform(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("GET to %s", req.RequestURI)
	platformID := getPlatformID(req)
	platformStorage := storage.Get().Platform()
	platform, err := platformStorage.GetByID(platformID)
	if err != nil {
		return getPlatformError(errorPlatformLookup(err), http.StatusInternalServerError)
	}
	if platform == nil {
		return getPlatformError(errorMissingPlatform(platformID), http.StatusNotFound)
	}
	if errJSON := rest.SendJSON(rw, http.StatusOK, platform); errJSON != nil {
		return getPlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
	}
	return nil
}

// getAllPlatforms handler for GET /v1/platforms
func (platformCtrl *Controller) getAllPlatforms(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("GET to %s", req.RequestURI)
	platformStorage := storage.Get().Platform()
	platforms, err := platformStorage.GetAll()
	if err != nil {
		return getAllPlatformsError(internalError(err, "could not get all platforms"), http.StatusInternalServerError)
	}
	platformsResponse := map[string][]rest.Platform{"platforms": platforms}

	if errJSON := rest.SendJSON(rw, http.StatusOK, &platformsResponse); errJSON != nil {
		return getAllPlatformsError(responseProcessingError(errJSON), http.StatusInternalServerError)
	}
	return nil
}

// deletePlatform handler for DELETE /v1/platforms/:platform_id
func (platformCtrl *Controller) deletePlatform(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("DELETE to %s", req.RequestURI)
	platformID := getPlatformID(req)

	platformStorage := storage.Get().Platform()
	errDelete := platformStorage.Delete(platformID)
	if errDelete == storage.MissingEntityError {
		return deletePlatformError(errorMissingPlatform(platformID), http.StatusNotFound)
	} else if errDelete != nil {
		return deletePlatformError(internalError(errDelete, "could not delete platform with id %s", platformID), http.StatusInternalServerError)
	}
	// map[string]string{} will result in empty JSON
	if errJSON := rest.SendJSON(rw, http.StatusOK, map[string]string{}); errJSON != nil {
		return deletePlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
	}
	return nil
}

// updatePlatform handler for PATCH /v1/platforms/:platform_id
func (platformCtrl *Controller) updatePlatform(rw http.ResponseWriter, req *http.Request) error {
	logrus.Debugf("PATCH to %s", req.RequestURI)
	platformID := getPlatformID(req)
	newPlatform, errDecode := getPlatformFromRequest(req)
	if errDecode != nil {
		return updatePlatformError(errorRequestBodyDecode(errDecode), http.StatusBadRequest)
	}

	platformStorage := storage.Get().Platform()
	platform, err := platformStorage.GetByID(platformID)
	if err != nil {
		return updatePlatformError(errorPlatformLookup(err), http.StatusInternalServerError)
	}
	if platform == nil {
		return updatePlatformError(errorMissingPlatform(platformID), http.StatusNotFound)
	}
	mergePlatforms(newPlatform, platform)

	errUpdate := platformStorage.Update(platform)
	if errUpdate == storage.MissingEntityError {
		return updatePlatformError(errorMissingPlatform(platformID), http.StatusNotFound)
	} else if errUpdate != nil {
		return updatePlatformError(errorSavePlatform(errUpdate), http.StatusInternalServerError)
	}
	if errJSON := rest.SendJSON(rw, http.StatusOK, platform); errJSON != nil {
		return updatePlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
	}
	return nil
}
