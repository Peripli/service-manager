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

import (
	"context"
	"encoding/json"
)

// ServiceAccess provides a way to add a hook for a platform specific way of enabling and disabling
// service and plan access.
//go:generate counterfeiter . ServiceAccess
type ServiceAccess interface {
	// EnableAccessForService enables the access to all plans of the service with the specified GUID
	// for the entities in the data
	EnableAccessForService(ctx context.Context, data json.RawMessage, serviceGUID string) error

	// EnableAccessForPlan enables the access to the plan with the specified GUID for
	// the entities in the data
	EnableAccessForPlan(ctx context.Context, data json.RawMessage, servicePlanGUID string) error

	// DisableAccessForService disables the access to all plans of the service with the specified GUID
	// for the entities in the data
	DisableAccessForService(ctx context.Context, data json.RawMessage, serviceGUID string) error

	// DisableAccessForPlan disables the access to the plan with the specified GUID for
	// the entities in the data
	DisableAccessForPlan(ctx context.Context, data json.RawMessage, servicePlanGUID string) error
}
