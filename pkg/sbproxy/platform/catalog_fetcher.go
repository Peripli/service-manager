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

import "context"

// CatalogFetcher provides a way to add a hook for platform specific way of refetching the service broker catalog on each
// run of the registration task. If the platform that this proxy represents already handles that, you don't
// have to implement this interface
//go:generate counterfeiter . CatalogFetcher
type CatalogFetcher interface {
	// Fetch contains the logic for platform specific catalog fetching for the provided service broker
	Fetch(ctx context.Context, serviceBroker *ServiceBroker) error
}
