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
	"github.com/Peripli/service-manager/api/base"
	"github.com/Peripli/service-manager/pkg/extension"

	"github.com/Peripli/service-manager/pkg/security"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

var _ web.Controller = &Controller{}

// Controller broker controller
type Controller struct {
	*base.Controller
}

type AdditionalInterceptors struct {
	CreateProviders []extension.CreateInterceptorProvider
	UpdateProviders []extension.UpdateInterceptorProvider
	DeleteProviders []extension.DeleteInterceptorProvider
}

//TODO if we can figure out a way to move the extra parameters and the default hook out of here, this can be generated
//TODO the end goal would be to define just the types.Broker and get a controller and database layer generated
func NewController(repository storage.Repository, encrypter security.Encrypter, osbClientCreateFunc osbc.CreateFunc, interceptors *AdditionalInterceptors) *Controller {
	//TODO not sure if passing the last argument is the best approach - probably .AttachHooks methods on the base.Controller that is embedded would be better
	//TODO this way we can move out the default hooks too and this file can be generated
	defaultCreateInterceptor := func() extension.CreateInterceptor {
		return &CreateBrokerHook{
			OSBClientCreateFunc: osbClientCreateFunc,
			Encrypter:           encrypter,
		}
	}
	createInterceptorProviders := append(interceptors.CreateProviders, defaultCreateInterceptor)
	createInterceptorProvider := extension.UnionCreateInterceptor(createInterceptorProviders...)

	defaultUpdateInterceptor := func() extension.UpdateInterceptor {
		return &UpdateBrokerHook{
			OSBClientCreateFunc: osbClientCreateFunc,
			Encrypter:           encrypter,
		}
	}
	updateInterceptorProviders := append(interceptors.UpdateProviders, defaultUpdateInterceptor)
	updateInterceptorProvider := extension.UnionUpdateInterceptor(updateInterceptorProviders...)

	return &Controller{
		Controller: &base.Controller{
			Repository:                repository,
			ObjectBlueprint:           func() types.Object { return &types.Broker{} },
			ObjectType:                types.BrokerType,
			ResourceBaseURL:           web.BrokersURL,
			CreateInterceptorProvider: createInterceptorProvider,
			UpdateInterceptorProvider: updateInterceptorProvider,
			DeleteInterceptorProvider: extension.UnionDeleteInterceptor(interceptors.DeleteProviders...),
		},
	}
}
