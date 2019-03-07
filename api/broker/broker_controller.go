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

package broker

import (
	"net/http"

	"github.com/Peripli/service-manager/api/base"

	"github.com/Peripli/service-manager/pkg/extension"

	"github.com/Peripli/service-manager/pkg/security"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const (
	reqBrokerID = "broker_id"
)

var _ web.Controller = &Controller{}

// Controller broker controller
type Controller struct {
	base.Controller
}

func NewController(repository storage.Repository, encrypter security.Encrypter, osbClientCreateFunc osbc.CreateFunc) *Controller {
	return &Controller{
		Controller: base.Controller{
			Repository:      repository,
			ObjectBlueprint: func() types.Object { return &types.Broker{} },
			ObjectType:      types.BrokerType,
			PathParamID:     reqBrokerID,
			CreateHookFunc: func(objectType types.ObjectType) extension.CreateHook {
				return &CreateBrokerHook{
					OSBClientCreateFunc: osbClientCreateFunc,
					Encrypter:           encrypter,
				}
			},
			UpdateHookFunc: func(objectType types.ObjectType) extension.UpdateHook {
				return &UpdateBrokerHook{
					OSBClientCreateFunc: osbClientCreateFunc,
					Encrypter:           encrypter,
				}
			},
		},
	}
}

// Routes returns slice of routes which handle broker operations
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   web.BrokersURL,
			},
			Handler: c.CreateObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.BrokersURL + "/{broker_id}",
			},
			Handler: c.GetSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.BrokersURL,
			},
			Handler: c.ListObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   web.BrokersURL,
			},
			Handler: c.DeleteObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   web.BrokersURL + "/{broker_id}",
			},
			Handler: c.DeleteSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   web.BrokersURL + "/{broker_id}",
			},
			Handler: c.PatchObject,
		},
	}
}
