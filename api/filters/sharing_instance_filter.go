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
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
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
	// Ignore the filter if has no shared property
	sharedBytes := gjson.GetBytes(req.Body, "shared")
	if len(sharedBytes.Raw) == 0 {
		return next.Handle(req)
	}

	ctx := req.Context()
	logger := log.C(ctx)
	body := map[string]bool{}
	util.BytesToObject(req.Body, body)
	shared := sharedBytes.Bool()

	if len(body) > 1 {
		return nil, errors.New("Could not modify the 'shared' property with other changes at the same time")
	}

	// Get instance from database
	instanceID := req.PathParams["resource_id"]
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	instanceObject, err := f.storageRepository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := instanceObject.(*types.ServiceInstance)

	if instance.Shared == shared {
		return nil, errors.New(fmt.Sprintf("The service instance is already set as shared=%s", sharedBytes))
	}

	// When un-sharing and has references
	if shared == false {
		referencesList, err := f.getInstanceReferencesByID(instance.ID)
		if err != nil {
			logger.Errorf("Could not retrieve references of the service instance (%s): %v", instanceID, err)
		}
		if referencesList.Len() > 0 {
			return nil, errors.New(fmt.Sprintf("Could not un-share the service instance. The service instance has %d references which should be deleted first", referencesList.Len()))
		}
	}

	// Get plan object from database, on service_instance patch flow
	planID := instance.ServicePlanID
	byID = query.ByField(query.EqualsOperator, "id", planID)
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

	return next.Handle(req)
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
