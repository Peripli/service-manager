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
	"errors"
	"fmt"
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

func (f *sharingInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	return f.handleServiceUpdate(req, next)

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

func (f *sharingInstanceFilter) validateOnlySharedPropertyIsChanged(persistedInstance *types.ServiceInstance, reqInstanceBytes *[]byte) error {
	var updatedInstance types.ServiceInstance
	persistedInstanceBytes, err := json.Marshal(&persistedInstance)
	if err != nil {
		return err
	}
	if err := util.BytesToObject(persistedInstanceBytes, &updatedInstance); err != nil {
		return err
	}
	if err := util.BytesToObject(*reqInstanceBytes, &updatedInstance); err != nil {
		return err
	}

	//in order to ignore shared property when validating the request we set it to be equals
	//TODO: find out why the context is not the same (the persisted instance has instance_name property and updatedInstance does not have it)
	updatedInstance.Shared = persistedInstance.Shared
	updatedInstance.Context = persistedInstance.Context

	if !persistedInstance.Equals(&updatedInstance) {
		return errors.New(fmt.Sprintf("Could not modify the 'shared' property with other changes at the same time"))
	}
	return nil
}

func (f *sharingInstanceFilter) getInstanceReferencesByID(instanceID string) (types.ObjectList, error) {
	references, err := f.storageRepository.List(
		context.Background(),
		types.ServiceInstanceType,
		query.ByField(query.EqualsOperator, "referenced_instance_id", instanceID))
	if err != nil {
		return nil, err
	}
	return references, nil
}

func (f *sharingInstanceFilter) handleServiceUpdate(req *web.Request, next web.Handler) (*web.Response, error) {
	var reqServiceInstance types.ServiceInstance
	var err error
	err = util.BytesToObjectNoLabels(req.Body, &reqServiceInstance)

	if err != nil {
		return nil, err
	}

	if reqServiceInstance.Shared == nil {
		return next.Handle(req)
	}

	ctx := req.Context()
	logger := log.C(ctx)

	// Get instance from database
	instanceID := req.PathParams["resource_id"]
	persistedInstance, err := f.retrievePersistedInstanceByID(ctx, instanceID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	//we cannot use reqServiceInstance in this validation because the struct has default values (like "" for string type properties)
	err = f.validateOnlySharedPropertyIsChanged(persistedInstance, &req.Body)
	if err != nil {
		return nil, err
	}

	// Get plan object from database, on service_instance patch flow
	planID := persistedInstance.ServicePlanID
	byID := query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.storageRepository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	if !plan.IsShareablePlan() {
		return nil, &util.UnsupportedQueryError{
			Message: "Plan is non-shared",
		}
	}

	if persistedInstance.Shared != nil && *persistedInstance.Shared == *reqServiceInstance.Shared {
		return nil, errors.New(fmt.Sprintf("The service instance is already at the desried state=%t", *reqServiceInstance.Shared))
	}

	// When un-sharing a service instance with references
	if persistedInstance.Shared != nil && !*reqServiceInstance.Shared {
		referencesList, err := f.getInstanceReferencesByID(persistedInstance.ID)

		if err != nil {
			logger.Errorf("Could not retrieve references of the service instance (%s): %v", instanceID, err)
		}

		if referencesList.Len() > 0 {
			errorMessage := fmt.Sprintf("could not delete the service instance. The service instance has %d references which should be deleted first", referencesList.Len())
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: errorMessage,
				StatusCode:  http.StatusBadRequest,
			}
		}
	}

	return next.Handle(req)
}

func (f *sharingInstanceFilter) retrievePersistedInstanceByID(ctx context.Context, instanceID string) (*types.ServiceInstance, error) {
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	dbInstanceObject, err := f.storageRepository.Get(ctx, types.ServiceInstanceType, byID)
	persistedInstance := dbInstanceObject.(*types.ServiceInstance)
	return persistedInstance, err
}
