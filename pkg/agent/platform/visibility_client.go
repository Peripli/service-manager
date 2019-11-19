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

	"github.com/Peripli/service-manager/pkg/types"
)

// Visibility generic visibility entity
type Visibility struct {
	Public             bool
	CatalogPlanID      string
	PlatformBrokerName string
	Labels             map[string]string
}

// ModifyPlanAccessRequest type used for requests by the platform client
type ModifyPlanAccessRequest struct {
	BrokerName    string       `json:"broker_name"`
	CatalogPlanID string       `json:"catalog_plan_id"`
	Labels        types.Labels `json:"labels"`
}

// VisibilityClient interface for platform clients to implement if they support
// platform specific service and plan visibilities
//go:generate counterfeiter . VisibilityClient
type VisibilityClient interface {
	// GetVisibilitiesByBrokers get currently available visibilities in the platform for specific broker names
	GetVisibilitiesByBrokers(context.Context, []string) ([]*Visibility, error)

	// VisibilityScopeLabelKey returns a specific label key which should be used when converting SM visibilities to platform.Visibilities
	VisibilityScopeLabelKey() string

	// EnableAccessForPlan enables the access for the specified plan
	EnableAccessForPlan(ctx context.Context, request *ModifyPlanAccessRequest) error

	// DisableAccessForPlan disables the access for the specified plan
	DisableAccessForPlan(ctx context.Context, request *ModifyPlanAccessRequest) error
}
