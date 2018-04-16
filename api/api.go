/*
 * Copyright 2018 The Service Manager Authors
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

 // Package api contains logic for building the Service Manager API business logic
package api

import (
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/rest/broker"
)

// Default returns the minimum set of REST APIs needed for the Service Manager
func Default(storage storage.Storage) rest.API {
	return &smAPI{
		controllers: []rest.Controller{
			broker.Controller{
				BrokerStorage: storage.Broker(),
			},
		},
	}
}

type smAPI struct {
	controllers []rest.Controller
}

func (api *smAPI) Controllers() []rest.Controller {
	return api.controllers
}

func (api *smAPI) RegisterControllers(controllers ...rest.Controller) {
	for _, controller := range controllers {
		if controller == nil {
			panic("Cannot add nil controllers")
		}
		api.controllers = append(api.controllers, controller)
	}
}
