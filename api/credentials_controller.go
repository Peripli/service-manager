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
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

// CredentialsController implements api.Controller by providing logic for broker platform credential storage/update
type CredentialsController struct {
	*BaseController
}

func NewCredentialsController(ctx context.Context, options *Options) *CredentialsController {
	return &CredentialsController{
		BaseController: NewController(ctx, options, web.CredentialsURL, types.BrokerPlatformCredentialType, func() types.Object {
			return &types.BrokerPlatformCredential{}
		}),
	}
}

func (c *CredentialsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   c.resourceBaseURL,
			},
			Handler: c.CreateObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   c.resourceBaseURL,
			},
			Handler: c.PatchObject,
		},
	}
}
