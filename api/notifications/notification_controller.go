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

package notifications

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/ws"

	"github.com/Peripli/service-manager/pkg/web"
)

// Controller implements api.Controller by providing service plans API logic
type Controller struct {
	baseCtx    context.Context
	repository storage.Repository

	wsSettings  *ws.Settings
	notificator storage.Notificator
}

// Routes returns the routes for notifications
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.NotificationsURL,
			},
			Handler: c.handleWS,
		},
	}
}

// NewController creates new notifications controller
func NewController(baseCtx context.Context, repository storage.Repository, wsSettings *ws.Settings, notificator storage.Notificator) *Controller {
	return &Controller{
		baseCtx:     baseCtx,
		repository:  repository,
		wsSettings:  wsSettings,
		notificator: notificator,
	}
}
