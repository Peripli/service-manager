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

	FetchCatalog(next Handler) Handler
}

// Provisioner should be implemented by plugins that need to intercept OSB call for provision operation
type Provisioner interface {
	Plugin

	Provision(next Handler) Handler
}

// Deprovisioner should be implemented by plugins that need to intercept OSB call for deprovision operation
type Deprovisioner interface {
	Plugin

	Deprovision(next Handler) Handler
}

// ServiceUpdater should be implemented by plugins that need to intercept OSB call for update service operation
type ServiceUpdater interface {
	Plugin

	UpdateService(next Handler) Handler
}

// ServiceFetcher should be implemented by plugins that need to intercept OSB call for get service operation
type ServiceFetcher interface {
	Plugin

	FetchService(next Handler) Handler
}

// Binder should be implemented by plugins that need to intercept OSB call for bind service operation
type Binder interface {
	Plugin

	Bind(next Handler) Handler
}

// Unbinder should be implemented by plugins that need to intercept OSB call for unbind service operation
type Unbinder interface {
	Plugin

	Unbind(next Handler) Handler
}

// BindingFetcher should be implemented by plugins that need to intercept OSB call for unbind service operation
type BindingFetcher interface {
	Plugin

	FetchBinding(next Handler) Handler
}
