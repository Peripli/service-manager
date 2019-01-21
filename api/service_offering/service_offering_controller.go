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

package service_offering

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const reqServiceOfferingID = "service_offering_id"

// Controller implements api.Controller by providing service offerings API logic
type Controller struct {
	ServiceOfferingStorage storage.ServiceOffering
}

func (c *Controller) getServiceOffering(r *web.Request) (*web.Response, error) {
	serviceOfferingID := r.PathParams[reqServiceOfferingID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting service offering with id %s", serviceOfferingID)

	serviceOffering, err := c.ServiceOfferingStorage.Get(ctx, serviceOfferingID)
	if err = util.HandleStorageError(err, "service_offering"); err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, serviceOffering)
}

func (c *Controller) listServiceOfferings(r *web.Request) (*web.Response, error) {
	var serviceOfferings []*types.ServiceOffering
	var err error
	ctx := r.Context()
	log.C(ctx).Debug("Listing service offerings")

	serviceOfferings, err = c.ServiceOfferingStorage.List(ctx, query.CriteriaForContext(ctx)...)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}

	return util.NewJSONResponse(http.StatusOK, struct {
		ServiceOfferings []*types.ServiceOffering `json:"service_offerings"`
	}{
		ServiceOfferings: serviceOfferings,
	})
}
