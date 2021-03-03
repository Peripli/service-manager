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
	"net/http"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const ReferenceBindingFilterName = "ReferenceBindingFilter"

// serviceBindingFilter ensures that the tenant making the provisioning/update request
// has the necessary visibilities - i.e. that he has the right to consume the requested plan.
type referenceBindingFilter struct {
	repository storage.Repository
}

func NewReferenceBindingFilter(repository storage.Repository) *referenceBindingFilter {
	return &referenceBindingFilter{
		repository: repository,
	}
}

func (*referenceBindingFilter) Name() string {
	return ReferenceBindingFilterName
}

func (f *referenceBindingFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {

	handle, err := next.Handle(req)

	return handle, err
	/*serviceInstanceID := gjson.GetBytes(req.Body, "service_instance_id").String()

	ctx := req.Context()

	byID := query.ByField(query.EqualsOperator, "id", serviceInstanceID)
	referenceInstanceObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	referenceInstance := referenceInstanceObject.(*types.ServiceInstance)

	sharedInstanceID := referenceInstance.ReferencedInstanceID
	byID = query.ByField(query.EqualsOperator, "id", sharedInstanceID)
	sharedInstanceObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	sharedInstance := sharedInstanceObject.(*types.ServiceInstance)
	fmt.Print(sharedInstance)
	return next.Handle(req)*/
	//return nil, nil
}

func (*referenceBindingFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBindingsURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch, http.MethodDelete),
			},
		},
	}
}
