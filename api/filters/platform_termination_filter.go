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
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
	"strings"
)

const (
	PlatformTerminationFilterName = "PlatformTerminationFilter"
)

func NewPlatformTerminationFilter(repository storage.Repository) *platformTerminationFilter {
	return &platformTerminationFilter{
		repository: repository,
	}
}

// platformTerminationFilter ensures that platform provided is considered as not active and only then deletion is possible.
type platformTerminationFilter struct {
	repository storage.Repository
}

func (*platformTerminationFilter) Name() string {
	return PlatformTerminationFilterName
}

func (f *platformTerminationFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	platformID := req.PathParams[web.PathParamResourceID]
	cascadeParam := req.URL.Query().Get(web.QueryParamCascade)
	if req.Request.Method == http.MethodDelete && platformID != "" && cascadeParam == "true" {
		ctx := req.Context()
		byID := query.ByField(query.EqualsOperator, "id", platformID)
		platformObject, err := f.repository.Get(ctx, types.PlatformType, byID)
		if err != nil {
			return nil, util.HandleStorageError(err, types.PlatformType.String())
		}
		platform := platformObject.(*types.Platform)
		if platform.Active {
			return nil, &util.HTTPError{
				ErrorType:   "UnprocessableEntity",
				Description: "Active platform cannot be deleted",
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}
		instancesInOtherPlatforms, err := findReferencesOfSharedInstancesInOtherPlatforms(ctx, platform, f.repository)
		if err != nil {
			return nil, err
		}

		if instancesInOtherPlatforms != nil && instancesInOtherPlatforms.Len() > 0 {
			return nil, &util.HTTPError{
				ErrorType:   "UnprocessableEntity",
				Description: "Platform cannot be deleted because other platform(s) has reference instance(s) to the shared instances in given platform. Details: " + formatSharingReferences(instancesInOtherPlatforms),
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}
	}
	return next.Handle(req)
}

func formatSharingReferences(references types.ObjectList) string {
	refBySharedInstanceID := make(map[string][]string)
	for i := 0; i < references.Len(); i++ {
		referenceInstance := references.ItemAt(i).(*types.ServiceInstance)
		sharedInstanceID := referenceInstance.ReferencedInstanceID
		refBySharedInstanceID[sharedInstanceID] = append(refBySharedInstanceID[sharedInstanceID], referenceInstance.GetID())
	}

	var msg []string
	for key, references := range refBySharedInstanceID {
		msg = append(msg, "shared instance "+key+" is referenced by instance(s)"+strings.Join(references, ", "))
	}
	return strings.Join(msg, ", ")
}

func findReferencesOfSharedInstancesInOtherPlatforms(ctx context.Context, platform *types.Platform, repository storage.Repository) (types.ObjectList, error) {
	sharedInstanceIDs, err := findSharedInstancesInPlatform(ctx, platform, repository)
	if err != nil {
		return nil, err
	}

	if len(sharedInstanceIDs) == 0 {
		return nil, nil
	}

	references, err := repository.ListNoLabels(ctx,
		types.ServiceInstanceType,
		query.ByField(query.InOperator, instance_sharing.ReferencedInstanceIDKey, sharedInstanceIDs...),
		query.ByField(query.NotEqualsOperator, "platform_id", platform.GetID()))

	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	return references, nil
}

func findSharedInstancesInPlatform(ctx context.Context, platform *types.Platform, repository storage.Repository) ([]string, error) {
	sharedInstances, err := repository.ListNoLabels(ctx, types.ServiceInstanceType,
		query.ByField(query.EqualsOperator, "platform_id", platform.ID),
		query.ByField(query.EqualsOperator, "shared", "true"),
	)

	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	return types.ObjectListIDsToStringArray(sharedInstances), nil
}

func (*platformTerminationFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.PlatformsURL + "/**"),
				web.Methods(http.MethodDelete),
			},
		},
	}
}
