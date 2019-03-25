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
	"github.com/Peripli/service-manager/api/base"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

type Controller struct {
	*base.Controller
}

var _ web.Controller = &Controller{}

func NewController(repository storage.Repository, encrypter security.Encrypter) *Controller {
	baseController := base.NewController(repository, web.PlatformsURL, func() types.Object {
		return &types.Platform{}
	})

	baseController.AddCreateInterceptorProviders(&createInterceptorProvider{
		encrypter: encrypter,
	})

	return &Controller{
		Controller: baseController,
	}
}
