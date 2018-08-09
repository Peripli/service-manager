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

package web

// Plugin can intercept Service Manager operations and augment them with additional logic.
// To intercept SM operations a plugin implements one or more of the interfaces defined in this package.
type Plugin interface {
	Named
}

// Interfaces for OSB operations

// CatalogFetcher should be implemented by plugins that need to intercept OSB call for get catalog operation
type CatalogFetcher interface {
	Plugin

	FetchCatalog(request *Request, next Handler) (*Response, error)
}

// Provisioner should be implemented by plugins that need to intercept OSB call for provision operation
type Provisioner interface {
	Plugin

	Provision(request *Request, next Handler) (*Response, error)
}

// Deprovisioner should be implemented by plugins that need to intercept OSB call for deprovision operation
type Deprovisioner interface {
	Plugin

	Deprovision(request *Request, next Handler) (*Response, error)
}

// ServiceUpdater should be implemented by plugins that need to intercept OSB call for update service operation
type ServiceUpdater interface {
	Plugin

	UpdateService(request *Request, next Handler) (*Response, error)
}

// ServiceFetcher should be implemented by plugins that need to intercept OSB call for get service operation
type ServiceFetcher interface {
	Plugin

	FetchService(request *Request, next Handler) (*Response, error)
}

// Binder should be implemented by plugins that need to intercept OSB call for bind service operation
type Binder interface {
	Plugin

	Bind(request *Request, next Handler) (*Response, error)
}

// Unbinder should be implemented by plugins that need to intercept OSB call for unbind service operation
type Unbinder interface {
	Plugin

	Unbind(request *Request, next Handler) (*Response, error)
}

// BindingFetcher should be implemented by plugins that need to intercept OSB call for unbind service operation
type BindingFetcher interface {
	Plugin

	FetchBinding(request *Request, next Handler) (*Response, error)
}

// InstancePoller should be implemented by plugins that need to intercept OSB calls for polling last operation for service instances
type InstancePoller interface {
	Plugin

	PollInstance(request *Request, next Handler) (*Response, error)
}

// BindingPoller should be implemented by plugins that need to intercept OSB Calls for polling last operation for service bindings
type BindingPoller interface {
	Plugin

	PollBinding(request *Request, next Handler) (*Response, error)
}
