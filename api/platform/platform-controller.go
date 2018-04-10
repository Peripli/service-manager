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

// "encoding/json"
// "errors"
// "fmt"
// "net/http"
// "time"

// "github.com/Peripli/service-manager/rest"
// "github.com/Peripli/service-manager/storage"
// "github.com/Peripli/service-manager/util"
// "github.com/Sirupsen/logrus"
// "github.com/gorilla/mux"

// Controller platform controller
type Controller struct{}

// func externalError(err error, message string, args ...interface{}) error {
// 	if err != nil {
// 		logrus.Debug(err.Error())
// 	}
// 	return fmt.Errorf(message, args)
// }

// func internalError(err error, message string, args ...interface{}) error {
// 	logrus.Error(err.Error())
// 	return fmt.Errorf(message, args)
// }

// func errorSavePlatform(reason error) error {
// 	return internalError(reason, "could not save platform")
// }

// func errorPlatformLookup(reason error) error {
// 	return internalError(reason, "error occurred during platform lookup")
// }

// func errorMissingPlatform(platformID string) error {
// 	return externalError(nil, "could not find platform with id %s", platformID)
// }

// func errorRequestBodyDecode(reason error) error {
// 	return externalError(reason, "error occurred while decoding request body")
// }

// func responseProcessingError(err error) error {
// 	return internalError(err, "Error while processing response")
// }

// func registerPlatformError(err error, statusCode int) error {
// 	return rest.CreateErrorResponse(err, statusCode, "RegisterPlatformError")
// }

// func getPlatformError(err error, statusCode int) error {
// 	return rest.CreateErrorResponse(err, statusCode, "GetPlatformError")
// }

// func getAllPlatformsError(err error, statusCode int) error {
// 	return rest.CreateErrorResponse(err, statusCode, "GetAllPlatformsError")
// }

// func deletePlatformError(err error, statusCode int) error {
// 	return rest.CreateErrorResponse(err, statusCode, "DeletePlatformError")
// }

// func updatePlatformError(err error, statusCode int) error {
// 	return rest.CreateErrorResponse(err, statusCode, "UpdatePlatformError")
// }

// func getPlatformID(req *http.Request) string {
// 	vars := mux.Vars(req)
// 	return vars["platform_id"]
// }

// func getPlatformFromRequest(req *http.Request) (*rest.Platform, error) {
// 	decoder := json.NewDecoder(req.Body)
// 	var platform rest.Platform
// 	if err := decoder.Decode(&platform); err != nil {
// 		return nil, err
// 	}
// 	return &platform, nil
// }

// func mergePlatforms(source *rest.Platform, target *dto.Platform) {
// 	if source.Name != "" {
// 		target.Name = source.Name
// 	} else {
// 		source.Name = target.Name
// 	}
// 	if source.Description != "" {
// 		target.Description = source.Description
// 	} else {
// 		source.Description = target.Description
// 	}
// 	if source.Type != "" {
// 		target.Type = source.Type
// 	} else {
// 		source.Type = target.Type
// 	}
// 	target.UpdatedAt = time.Now()
// 	source.UpdatedAt = util.ToRFCFormat(target.UpdatedAt)
// 	source.CreatedAt = util.ToRFCFormat(target.CreatedAt)
// 	source.ID = target.ID
// }

// func checkPlatformMandatoryProperties(platform *rest.Platform) error {
// 	if platform.Type == "" {
// 		return errors.New("missing platform type")
// 	}
// 	if platform.Name == "" {
// 		return errors.New("missing platform name")
// 	}
// 	return nil
// }

// func restPlatformFromDTO(platformDTO *dto.Platform) *rest.Platform {
// 	return &rest.Platform{
// 		ID:          platformDTO.ID,
// 		Type:        platformDTO.Type,
// 		Name:        platformDTO.Name,
// 		Description: platformDTO.Description,
// 		CreatedAt:   util.ToRFCFormat(platformDTO.CreatedAt),
// 		UpdatedAt:   util.ToRFCFormat(platformDTO.UpdatedAt)}
// }

// // addPlatform handler for POST /v1/platforms
// func (platformCtrl *Controller) addPlatform(rw http.ResponseWriter, req *http.Request) error {
// 	logrus.Debug("POST to %s", req.RequestURI)

// 	platform, errDecode := getPlatformFromRequest(req)
// 	if errDecode != nil {
// 		return registerPlatformError(errorRequestBodyDecode(errDecode), http.StatusBadRequest)
// 	}
// 	if errMandatoryProperties := checkPlatformMandatoryProperties(platform); errMandatoryProperties != nil {
// 		logrus.Debug(errMandatoryProperties)
// 		return registerPlatformError(errMandatoryProperties, http.StatusBadRequest)
// 	}
// 	platformStorage := storage.Get().Platform()
// 	if platform.ID != "" {
// 		platformFromDB, err := platformStorage.GetByID(platform.ID)
// 		if err != nil {
// 			return registerPlatformError(errorPlatformLookup(err), http.StatusInternalServerError)
// 		}
// 		if platformFromDB != nil {
// 			return registerPlatformError(externalError(nil, "platform with id %s already exists", platform.ID), http.StatusConflict)
// 		}
// 	}
// 	platformWithSameName, errNameCollision := platformStorage.GetByName(platform.Name)
// 	if errNameCollision != nil {
// 		return registerPlatformError(errorPlatformLookup(errNameCollision), http.StatusInternalServerError)
// 	}
// 	if platformWithSameName != nil {
// 		return registerPlatformError(externalError(nil, "platform with name %s already exists", platform.Name), http.StatusConflict)
// 	}

// 	username, password := util.GenerateCredentials()
// 	if platform.ID == "" {
// 		platform.ID = util.GenerateID()
// 	}

// 	currentTime := time.Now().UTC()

// 	platform.CreatedAt = util.ToRFCFormat(currentTime)
// 	platform.UpdatedAt = util.ToRFCFormat(currentTime)
// 	platform.Credentials = &rest.Credentials{
// 		Basic: &rest.Basic{
// 			Username: username,
// 			Password: password,
// 		},
// 	}

// 	platformDTO := &dto.Platform{
// 		ID:          platform.ID,
// 		Type:        platform.Type,
// 		Name:        platform.Name,
// 		Description: platform.Description,
// 		CreatedAt:   currentTime,
// 		UpdatedAt:   currentTime,
// 	}
// 	credentialsDTO := &dto.Credentials{
// 		Username: username,
// 		Password: password,
// 	}

// 	if errSave := platformStorage.Create(platformDTO, credentialsDTO); errSave != nil {
// 		return registerPlatformError(errorSavePlatform(errSave), http.StatusInternalServerError)
// 	}
// 	if errJSON := util.SendJSON(rw, http.StatusCreated, platform); errJSON != nil {
// 		return registerPlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
// 	}
// 	return nil
// }

// // getPlatform handler for GET /v1/platforms/:platform_id
// func (platformCtrl *Controller) getPlatform(rw http.ResponseWriter, req *http.Request) error {
// 	logrus.Debug("GET to %s", req.RequestURI)
// 	platformID := getPlatformID(req)
// 	platformStorage := storage.Get().Platform()
// 	platformDTO, err := platformStorage.GetByID(platformID)
// 	if err != nil {
// 		return getPlatformError(errorPlatformLookup(err), http.StatusInternalServerError)
// 	}
// 	if platformDTO == nil {
// 		return getPlatformError(errorMissingPlatform(platformID), http.StatusNotFound)
// 	}
// 	errJSON := util.SendJSON(rw, http.StatusOK, restPlatformFromDTO(platformDTO))

// 	if errJSON != nil {
// 		return getPlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
// 	}
// 	return nil
// }

// // getAllPlatforms handler for GET /v1/platforms
// func (platformCtrl *Controller) getAllPlatforms(rw http.ResponseWriter, req *http.Request) error {
// 	platformStorage := storage.Get().Platform()
// 	platformDTOs, err := platformStorage.GetAll()
// 	if err != nil {
// 		logrus.Error(err.Error())
// 		return getAllPlatformsError(errors.New("Could not get all platforms"), http.StatusInternalServerError)
// 	}
// 	var platforms = make([]*rest.Platform, 0, len(platformDTOs))
// 	for _, platformDTO := range platformDTOs {
// 		platforms = append(platforms, restPlatformFromDTO(&platformDTO))
// 	}
// 	platformsResponse := map[string][]*rest.Platform{"platforms": platforms}

// 	if errJSON := util.SendJSON(rw, http.StatusOK, &platformsResponse); errJSON != nil {
// 		return getAllPlatformsError(responseProcessingError(errJSON), http.StatusInternalServerError)
// 	}
// 	return nil
// }

// // deletePlatform handler for DELETE /v1/platforms/:platform_id
// func (platformCtrl *Controller) deletePlatform(rw http.ResponseWriter, req *http.Request) error {
// 	logrus.Debug("DELETE to %s", req.RequestURI)
// 	platformID := getPlatformID(req)
// 	platformStorage := storage.Get().Platform()
// 	platformDTO, err := platformStorage.GetByID(platformID)
// 	if err != nil {
// 		return deletePlatformError(errorPlatformLookup(err), http.StatusInternalServerError)
// 	}
// 	if platformDTO == nil {
// 		return deletePlatformError(errorMissingPlatform(platformID), http.StatusNotFound)
// 	}
// 	errDelete := platformStorage.Delete(platformID)
// 	if errDelete != nil {
// 		logrus.Error(errDelete)
// 		return deletePlatformError(fmt.Errorf("Could not delete platform with id %s", platformID), http.StatusInternalServerError)
// 	}
// 	// map[string]string{} will result in empty JSON
// 	if errJSON := util.SendJSON(rw, http.StatusOK, map[string]string{}); errJSON != nil {
// 		return deletePlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
// 	}
// 	return nil
// }

// // updatePlatform handler for PUT /v1/platforms/:platform_id
// func (platformCtrl *Controller) updatePlatform(rw http.ResponseWriter, req *http.Request) error {
// 	logrus.Debug("PUT to %s", req.RequestURI)
// 	platformID := getPlatformID(req)
// 	platformStorage := storage.Get().Platform()
// 	platformDTO, err := platformStorage.GetByID(platformID)
// 	if err != nil {
// 		return updatePlatformError(errorPlatformLookup(err), http.StatusInternalServerError)
// 	}
// 	if platformDTO == nil {
// 		return updatePlatformError(errorMissingPlatform(platformID), http.StatusNotFound)
// 	}
// 	newPlatform, errDecode := getPlatformFromRequest(req)
// 	if errDecode != nil {
// 		return updatePlatformError(errorRequestBodyDecode(errDecode), http.StatusBadRequest)
// 	}
// 	if platformDTO.Name != newPlatform.Name {
// 		platformWithSameName, errNameCollisionCheck := platformStorage.GetByName(newPlatform.Name)
// 		if errNameCollisionCheck != nil {
// 			logrus.Error(errNameCollisionCheck)
// 			return updatePlatformError(errors.New("Error during platform update"), http.StatusInternalServerError)
// 		}
// 		if platformWithSameName != nil {
// 			return updatePlatformError(fmt.Errorf("platform with name %s already exists", newPlatform.Name), http.StatusConflict)
// 		}
// 	}
// 	mergePlatforms(newPlatform, platformDTO)

// 	if errUpdate := platformStorage.Update(platformDTO); errUpdate != nil {
// 		return updatePlatformError(errorSavePlatform(errUpdate), http.StatusInternalServerError)
// 	}
// 	if errJSON := util.SendJSON(rw, http.StatusOK, newPlatform); errJSON != nil {
// 		return updatePlatformError(responseProcessingError(errJSON), http.StatusInternalServerError)
// 	}
// 	return nil
// }
