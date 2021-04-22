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
	"github.com/Peripli/service-manager/constant"
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
		sharingReferences, err := findSharingReferenceInstancesFromOtherPlatform(ctx, platform, f.repository)
		if err != nil {
			return nil, err
		}
		if len(sharingReferences) > 0 {
			return nil, &util.HTTPError{
				ErrorType:   "UnprocessableEntity",
				Description: "Platform cannot be deleted because other platform(s) has reference instance(s) to the shared instances in given platform. Details: " + formatSharingReferences(sharingReferences),
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}
	}
	return next.Handle(req)
}

func formatSharingReferences(references []*SharingReferences) string {
	var msg []string
	for i := 0; i < len(references); i++ {
		msg = append(msg, "shared instance "+references[i].sharedInstanceID+" referenced by instance(s) "+strings.Join(references[i].referenceIDs, ", "))
	}
	return strings.Join(msg, ", ")
}

type SharingReferences struct {
	sharedInstanceID string
	referenceIDs     []string
}

func findSharingReferenceInstancesFromOtherPlatform(ctx context.Context, platform *types.Platform, repository storage.Repository) ([]*SharingReferences, error) {
	sharedInstances, err := repository.ListNoLabels(ctx, types.ServiceInstanceType,
		query.ByField(query.EqualsOperator, "platform_id", platform.ID),
		query.ByField(query.EqualsOperator, "shared", "true"),
	)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	var result []*SharingReferences
	for i := 0; i < sharedInstances.Len(); i++ {
		sharedInstance := sharedInstances.ItemAt(i).(*types.ServiceInstance)
		references, err := repository.ListNoLabels(
			context.Background(),
			types.ServiceInstanceType,
			query.ByField(query.EqualsOperator, constant.ReferencedInstanceIDKey, sharedInstance.GetID()),
			query.ByField(query.NotEqualsOperator, "platform_id", platform.GetID()),
		)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}
		if references.Len() > 0 {
			var referenceIDs []string
			for ri := 0; ri < references.Len(); ri++ {
				referenceIDs = append(referenceIDs, references.ItemAt(ri).GetID())
			}
			result = append(result, &SharingReferences{
				sharedInstanceID: sharedInstance.GetID(),
				referenceIDs:     referenceIDs,
			})
		}
	}
	return result, nil
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
