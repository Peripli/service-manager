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

package filters

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"net/http"
)

const SharedInstanceUpdateFilterName = "SharedInstanceUpdateFilter"

// SharingInstanceFilter validate the un/share request on an existing service instance
type sharedInstanceUpdateFilter struct {
	storageRepository storage.Repository
}

func NewSharedInstanceUpdateFilter(storageRepository storage.Repository) *sharedInstanceUpdateFilter {
	return &sharedInstanceUpdateFilter{
		storageRepository: storageRepository,
	}
}

func (*sharedInstanceUpdateFilter) Name() string {
	return SharedInstanceUpdateFilterName
}

func (sf *sharedInstanceUpdateFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	var reqServiceInstance types.ServiceInstance
	err := util.BytesToObjectNoLabels(req.Body, &reqServiceInstance)
	if err != nil {
		log.C(req.Context()).Errorf("Failed to parse the request body to an instance object: %s", err)
		return nil, err
	}

	switch req.Request.Method {
	case http.MethodPost:
		return sf.handleProvision(req, reqServiceInstance, next)
	case http.MethodPatch:
		return sf.handleServiceUpdate(req, reqServiceInstance, next)
	}
	return next.Handle(req)
}

func (*sharedInstanceUpdateFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPatch, http.MethodPost),
			},
		},
	}
}

func (sf *sharedInstanceUpdateFilter) handleProvision(req *web.Request, reqServiceInstance types.ServiceInstance, next web.Handler) (*web.Response, error) {
	// we don't allow setting the shared property while provisioning the instance - supported by the patch instance only.
	if reqServiceInstance.Shared != nil {
		log.C(req.Context()).Errorf("Failed to provision the instance, request body should not contain a 'shared' property")
		return nil, util.HandleInstanceSharingError(util.ErrInvalidProvisionRequestWithSharedProperty, "")
	}
	return next.Handle(req)
}

func (sf *sharedInstanceUpdateFilter) handleServiceUpdate(req *web.Request, reqServiceInstance types.ServiceInstance, next web.Handler) (*web.Response, error) {
	instanceID := req.PathParams["resource_id"]
	ctx := req.Context()

	// get instance from database
	dbPersistedInstanceObject, err := storage.GetObjectByField(ctx, sf.storageRepository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	persistedInstance := dbPersistedInstanceObject.(*types.ServiceInstance)

	// if changing plan of a shared instance, validate the new plan supports instance sharing.
	if persistedInstance.IsShared() && isPlanChanged(persistedInstance, reqServiceInstance) {
		dbNewPlanObject, err := storage.GetObjectByField(ctx, sf.storageRepository, types.ServicePlanType, "id", reqServiceInstance.ServicePlanID)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServicePlanType.String())
		}
		newPlan := dbNewPlanObject.(*types.ServicePlan)
		if !newPlan.SupportsInstanceSharing() {
			return nil, util.HandleInstanceSharingError(util.ErrNewPlanDoesNotSupportInstanceSharing, "")
		}
	}

	if reqServiceInstance.Shared == nil {
		return next.Handle(req)
	}

	// async flow is not supported when sharing instances:
	isAsync := req.URL.Query().Get(web.QueryParamAsync)
	if isAsync == "true" {
		return nil, util.HandleInstanceSharingError(util.ErrAsyncNotSupportedForSharing, instanceID)
	}

	logger := log.C(ctx)

	// we cannot use reqServiceInstance in this validation because the struct has default values (like "" for string type properties)
	err = validateRequestContainsSingleProperty(logger, req.Body, instanceID)
	if err != nil {
		log.C(ctx).Errorf("Failed to validate the request, request body should contain a single property: %s", err)
		return nil, err
	}

	// Get plan object from database, on service_instance patch flow
	dbPlanObject, err := storage.GetObjectByField(ctx, sf.storageRepository, types.ServicePlanType, "id", persistedInstance.ServicePlanID)
	if err != nil {
		log.C(ctx).Errorf("Failed to retrieve %s with id: %s, err: %s", types.ServicePlanType, persistedInstance.ServicePlanID, err)
		return nil, err
	}
	plan := dbPlanObject.(*types.ServicePlan)

	if !plan.SupportsInstanceSharing() {
		log.C(ctx).Errorf("The plan: %s does not support instance sharing", plan.ID)
		return nil, util.HandleInstanceSharingError(util.ErrPlanDoesNotSupportInstanceSharing, plan.ID)
	}

	if persistedInstance.IsShared() == *reqServiceInstance.Shared {
		return util.NewJSONResponse(http.StatusOK, persistedInstance)
	}

	// When un-sharing a service instance with references (validate has no references)
	if persistedInstance.IsShared() && !*reqServiceInstance.Shared {
		referencesList, err := storage.GetInstanceReferencesByID(ctx, sf.storageRepository, persistedInstance.ID)
		if err != nil {
			logger.Errorf("Could not retrieve references of the service instance (%s): %v", instanceID, err)
		}
		if referencesList != nil && referencesList.Len() > 0 {
			referencesArray := types.ObjectListIDsToStringArray(referencesList)
			log.C(req.Context()).Errorf("Failed to unshare the instance: %s due to existing references: %s", persistedInstance.ID, referencesArray)
			return nil, util.HandleReferencesError(util.ErrUnsharingInstanceWithReferences, referencesArray)
		}
	}

	log.C(ctx).Infof("Reference Instance Update passed successfully. InstanceID: \"%s\"", instanceID)
	return next.Handle(req)
}

func validateRequestContainsSingleProperty(logger *logrus.Entry, reqInstanceBytes []byte, instanceID string) error {
	var reqAsMap map[string]interface{}
	err := json.Unmarshal(reqInstanceBytes, &reqAsMap)
	if err != nil {
		logger.Errorf("Failed to unmarshal request for the instance %s on 'validateRequestContainsSingleProperty':\n%s", instanceID, reqInstanceBytes)
		return err
	}

	if len(reqAsMap) > 1 {
		return util.HandleInstanceSharingError(util.ErrInvalidShareRequest, instanceID)
	}

	return nil
}

func isPlanChanged(persistedInstance *types.ServiceInstance, reqServiceInstance types.ServiceInstance) bool {
	return reqServiceInstance.ServicePlanID != "" && persistedInstance.ServicePlanID != reqServiceInstance.ServicePlanID
}
