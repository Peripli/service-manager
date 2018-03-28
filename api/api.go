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

package api

import (
	"github.com/Peripli/service-manager/rest"
)

// Default returns the minimum set of REST APIs needed for the Service Manager
func Default() rest.Api {
	return &defaultAPI{}
}

type defaultAPI struct {
	controllers []rest.Controller
}

func (api *defaultAPI) Controllers() []rest.Controller {
	return api.controllers
}

func (api *defaultAPI) RegisterControllers(controllers ...rest.Controller) {
	for _, controller := range controllers {
		if controller == nil {
			panic("Cannot add nil controllers")
		}
		api.controllers = append(api.controllers, controller)
	}
}
