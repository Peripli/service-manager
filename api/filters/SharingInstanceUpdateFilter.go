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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
	"net/http"
)

const SharingInstanceUpdateFilterName = "SharingInstanceUpdateFilter"

// SharingInstanceFilter validate the un/share request on an existing service instance
type sharingInstanceUpdateFilter struct {
	storageRepository storage.Repository
}

func NewSharingInstanceUpdateFilter(storageRepository storage.Repository) *sharingInstanceUpdateFilter {
	return &sharingInstanceUpdateFilter{
		storageRepository: storageRepository,
	}
}

func (*sharingInstanceUpdateFilter) Name() string {
	return SharingInstanceUpdateFilterName
}

func (sf *sharingInstanceUpdateFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	var reqServiceInstance types.ServiceInstance
	err := util.BytesToObjectNoLabels(req.Body, &reqServiceInstance)
	if err != nil {
		return nil, err
	}

	ctx := req.Context()
	instanceID := req.PathParams["resource_id"]

	// Get instance from database
	dbPersistedInstanceObject, err := storage.GetObjectByField(ctx, sf.storageRepository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	persistedInstance := dbPersistedInstanceObject.(*types.ServiceInstance)
	if changingPlanOfSharedInstance(persistedInstance, reqServiceInstance) {
		// we don't allow changing plan of shared instance
		return nil, util.HandleInstanceSharingError(util.ErrChangingPlanOfSharedInstance, persistedInstance.ID)
	}
	if reqServiceInstance.Shared == nil {
		return next.Handle(req)
	}

	isAsync := req.URL.Query().Get(web.QueryParamAsync)
	if isAsync == "true" {
		return nil, util.HandleInstanceSharingError(util.ErrAsyncNotSupportedForSharing, instanceID)
	}

	logger := log.C(ctx)

	//we cannot use reqServiceInstance in this validation because the struct has default values (like "" for string type properties)
	err = validateRequestContainsSingleProperty(logger, req.Body, instanceID)
	if err != nil {
		return nil, err
	}

	// Get plan object from database, on service_instance patch flow
	dbPlanObject, err := storage.GetObjectByField(ctx, sf.storageRepository, types.ServicePlanType, "id", persistedInstance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	plan := dbPlanObject.(*types.ServicePlan)

	if !plan.IsShareablePlan() {
		return nil, &util.UnsupportedQueryError{
			Message: "Plan is non-shared",
		}
	}

	if persistedInstance.IsShared() == *reqServiceInstance.Shared {
		return nil, util.HandleInstanceSharingError(util.ErrInstanceIsAlreadyAtDesiredSharedState, persistedInstance.ID)
	}

	// When un-sharing a service instance with references
	if persistedInstance.IsShared() && !*reqServiceInstance.Shared {
		referencesList, err := storage.GetInstanceReferencesByID(ctx, sf.storageRepository, persistedInstance.ID)

		if err != nil {
			logger.Errorf("Could not retrieve references of the service instance (%s): %v", instanceID, err)
		}

		if referencesList.Len() > 0 {
			return nil, util.HandleReferencesError(util.ErrUnsharingInstanceWithReferences, types.ObjectListIDsToStringArray(referencesList))
		}
	}

	log.C(ctx).Infof("Reference Instance Update passed successfully. InstanceID: \"%s\"", instanceID)
	return next.Handle(req)
}

func changingPlanOfSharedInstance(persistedInstance *types.ServiceInstance, reqServiceInstance types.ServiceInstance) bool {
	return persistedInstance.IsShared() && reqServiceInstance.ServicePlanID != "" && persistedInstance.ServicePlanID != reqServiceInstance.ServicePlanID
}

func (*sharingInstanceUpdateFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPatch),
			},
		},
	}
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
