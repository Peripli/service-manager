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
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

const SharingInstanceFilterName = "SharingInstanceFilter"

// SharingInstanceFilter validate the un/share request on an existing service instance
type sharingInstanceFilter struct {
	storageRepository storage.Repository
}

func NewSharingInstanceFilter(storageRepository storage.Repository) *sharingInstanceFilter {
	return &sharingInstanceFilter{
		storageRepository: storageRepository,
	}
}

func (*sharingInstanceFilter) Name() string {
	return SharingInstanceFilterName
}

func (sf *sharingInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	return sf.handleServiceUpdate(req, next)
}

func (*sharingInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}

func validateRequestContainsSingleProperty(reqInstanceBytes []byte, instanceID string) error {
	var reqAsMap map[string]interface{}
	err := json.Unmarshal(reqInstanceBytes, &reqAsMap)

	if err != nil {
		return err
	}

	if len(reqAsMap) > 1 {
		return util.HandleInstanceSharingError(util.ErrInvalidShareRequest, instanceID)
	}

	return nil
}

func (sf *sharingInstanceFilter) getInstanceReferencesByID(ctx context.Context, instanceID string) (types.ObjectList, error) {
	references, err := sf.storageRepository.List(
		ctx,
		types.ServiceInstanceType,
		query.ByField(query.EqualsOperator, instance_sharing.ReferencedInstanceIDKey, instanceID))
	if err != nil {
		return nil, err
	}
	return references, nil
}

func (sf *sharingInstanceFilter) handleServiceUpdate(req *web.Request, next web.Handler) (*web.Response, error) {
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
		return nil, err
	}
	persistedInstance := dbPersistedInstanceObject.(*types.ServiceInstance)

	// we don't allow changing plan of shared instance
	if persistedInstance.IsShared() && persistedInstance.ServicePlanID != reqServiceInstance.ServicePlanID && reqServiceInstance.Shared == nil {
		return nil, util.HandleInstanceSharingError(util.ErrChangingPlanOfSharedInstance, persistedInstance.ID)
	}

	if reqServiceInstance.Shared == nil {
		return next.Handle(req)
	}

	//we cannot use reqServiceInstance in this validation because the struct has default values (like "" for string type properties)
	err = validateRequestContainsSingleProperty(req.Body, instanceID)
	if err != nil {
		return nil, err
	}

	logger := log.C(ctx)

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
	if !persistedInstance.IsShared() {
		referencesList, err := sf.getInstanceReferencesByID(ctx, persistedInstance.ID)

		if err != nil {
			logger.Errorf("Could not retrieve references of the service instance (%s): %v", instanceID, err)
		}

		if referencesList.Len() > 0 {
			return nil, util.HandleReferencesError(util.ErrSharedInstanceHasReferences, types.ObjectListIDsToStringArray(referencesList))
		}
	}

	log.C(ctx).Infof("Reference Instance Update passed successfully. InstanceID: \"%s\"", instanceID)
	return next.Handle(req)
}
