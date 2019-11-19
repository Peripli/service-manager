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

package platform

// Client is an interface for service related operations on a platform.
// If a platform does not support some operations they should
// return nil for the specific client.
//go:generate counterfeiter . Client
type Client interface {
	// Broker returns a BrokerClient which handles platform specific broker operations
	Broker() BrokerClient
	// Visibility returns a VisibilityClient which handles platform specific service visibility operations
	Visibility() VisibilityClient
	// CatalogFetcher returns a CatalogFetcher which handles platform specific fetching of service catalogs
	CatalogFetcher() CatalogFetcher
}
