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

package api

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

// NotificationController implements api.Controller by providing service plans API logic
type NotificationController struct {
}

// Routes returns the routes for notifications
func (c *NotificationController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.NotificationsURL,
			},
			Handler: func(req *web.Request) (resp *web.Response, err error) {
				return nil, &util.HTTPError{
					StatusCode:  http.StatusNotImplemented,
					Description: "Not Implemented",
					ErrorType:   "Not Implemented",
				}
			},
		},
	}
}

// TODO: create the actual websocket handling and disable CRUD and List operations
func NewNotificationController() *NotificationController {
	return &NotificationController{}
}
