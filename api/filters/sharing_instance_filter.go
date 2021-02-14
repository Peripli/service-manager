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
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

const SharingInstanceFilterName = "SharingInstanceFilter"

// ServiceInstanceStripFilter checks post/patch request body for unmodifiable properties
type sharingInstanceFilter struct {
	repository storage.TransactionalRepository
}

func NewSharingInstanceFilter(repository storage.TransactionalRepository) *sharingInstanceFilter {
	return &sharingInstanceFilter{
		repository: repository,
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

	instanceID := req.PathParams["resource_id"]
	shared := sharedBytes.Bool()

	// Get instance from database
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	instanceObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := instanceObject.(*types.ServiceInstance)

	if isSMPlatform(instance.PlatformID) {
		if req.Body, err = sjson.DeleteBytes(req.Body, "shared"); err != nil {
			return nil, err
		}
	} else if len(req.Body) > 1 {
		return nil, errors.New("could not modify the 'shared' property with other changes at the same time")
	}

	planID := instance.ServicePlanID
	// get plan object from database, on service_instance patch flow
	byID = query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	if shared && !plan.IsShareablePlan() {
		return nil, &util.UnsupportedQueryError{
			Message: "Plan is non-shared",
		}
	}

	if plan.IsShareablePlan() {
		err = f.shareInstance(instance, shared, err, ctx, logger, instanceID)
		// todo: return error to client
		if err != nil {
			logger.Errorf("Could not update shared property for instance (%s): %v", instanceID, err)
			return nil, err
		}

		err = f.setVisibilityLabelOfReferencePlan()
		if err != nil {
			logger.Errorf("Could not set a visibility label of reference plan when sharing the instance (%s): %v", instanceID, err)
			return nil, err
		}
	}

	return next.Handle(req)
}

func (f *sharingInstanceFilter) shareInstance(instance *types.ServiceInstance, shared bool, err error, ctx context.Context, logger *logrus.Entry, instanceID string) error {
	instance.Shared = shared
	err = f.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		_, err := storage.Update(ctx, instance, nil)
		if err != nil {
			logger.Errorf("Could not update shared property for instance (%s): %v", instanceID, err)
			return err
		}
		return nil
	})
	return err
}

func isSMPlatform(platformID string) bool {
	return platformID == types.SMPlatform

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

func (f *sharingInstanceFilter) setVisibilityLabelOfReferencePlan() error {
	return nil
}
