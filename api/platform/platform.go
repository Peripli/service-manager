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

package platform

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/rest"
	"github.com/gorilla/mux"
)

const (
	ApiVersion = "v1"
	Root       = "platforms"
	Url        = "/" + ApiVersion + "/" + Root
)

type Controller struct{}

func (platformCtrl *Controller) Routes() []rest.Route {
	return []rest.Route{
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodPost,
				Path:   Url,
			},
			Handler: platformCtrl.addPlatform,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodGet,
				Path:   Url + "/{platform_id}",
			},
			Handler: platformCtrl.getPlatform,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodGet,
				Path:   Url,
			},
			Handler: platformCtrl.getAllPlatforms,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodDelete,
				Path:   Url + "/{platform_id}",
			},
			Handler: platformCtrl.deletePlatform,
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodPatch,
				Path:   Url + "/{platform_id}",
			},
			Handler: platformCtrl.updatePlatform,
		},
	}
}

func (platformCtrl *Controller) addPlatform(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (platformCtrl *Controller) getPlatform(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Platform id: %v\n", vars["platform_id"])
	return nil
}

func (platformCtrl *Controller) getAllPlatforms(w http.ResponseWriter, r *http.Request) error {
	w.Write([]byte("No platforms available."))
	return nil
}

func (platformCtrl *Controller) deletePlatform(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (platformCtrl *Controller) updatePlatform(w http.ResponseWriter, r *http.Request) error {
	return nil
}
