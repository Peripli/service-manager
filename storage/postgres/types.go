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

package postgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

const (
	// platformTable db table name for platforms
	platformTable = "platforms"

	// brokerTable db table name for brokers
	brokerTable = "brokers"

	// serviceOfferingTable db table for service offerings
	serviceOfferingTable = "service_offerings"

	// servicePlanTable db table for service plans
	servicePlanTable = "service_plans"

	// visibilityTable db table for visibilities
	visibilityTable = "visibilities"
)

// Safe represents a secret entity
type Safe struct {
	Secret    []byte    `db:"secret"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Platform entity
type Platform struct {
	ID          string         `db:"id"`
	Type        string         `db:"type"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	Username    string         `db:"username"`
	Password    string         `db:"password"`
}

// Broker entity
type Broker struct {
	ID          string         `db:"id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	BrokerURL   string         `db:"broker_url"`
	Username    string         `db:"username"`
	Password    string         `db:"password"`
}

type ServiceOffering struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`

	Bindable             bool   `db:"bindable"`
	InstancesRetrievable bool   `db:"instances_retrievable"`
	BindingsRetrievable  bool   `db:"bindings_retrievable"`
	PlanUpdatable        bool   `db:"plan_updateable"`
	CatalogID            string `db:"catalog_id"`
	CatalogName          string `db:"catalog_name"`

	Tags     sqlxtypes.JSONText `db:"tags"`
	Requires sqlxtypes.JSONText `db:"requires"`
	Metadata sqlxtypes.JSONText `db:"metadata"`

	BrokerID string `db:"broker_id"`
}

type ServicePlan struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`

	Free          bool   `db:"free"`
	Bindable      bool   `db:"bindable"`
	PlanUpdatable bool   `db:"plan_updateable"`
	CatalogID     string `db:"catalog_id"`
	CatalogName   string `db:"catalog_name"`

	Metadata sqlxtypes.JSONText `db:"metadata"`
	Schemas  sqlxtypes.JSONText `db:"schemas"`

	ServiceOfferingID string `db:"service_offering_id"`
}

type Visibility struct {
	ID            string         `db:"id"`
	PlatformID    sql.NullString `db:"platform_id"`
	ServicePlanID string         `db:"service_plan_id"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}

func (b *Broker) ToDTO() *types.Broker {
	broker := &types.Broker{
		ID:          b.ID,
		Name:        b.Name,
		Description: b.Description.String,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		BrokerURL:   b.BrokerURL,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: b.Username,
				Password: b.Password,
			},
		},
	}
	return broker
}

func (b *Broker) FromDTO(broker *types.Broker) {
	*b = Broker{
		ID:          broker.ID,
		Description: sql.NullString{String: broker.Description},
		Name:        broker.Name,
		BrokerURL:   broker.BrokerURL,
		CreatedAt:   broker.CreatedAt,
		UpdatedAt:   broker.UpdatedAt,
	}

	if broker.Description != "" {
		b.Description.Valid = true
	}
	if broker.Credentials != nil && broker.Credentials.Basic != nil {
		b.Username = broker.Credentials.Basic.Username
		b.Password = broker.Credentials.Basic.Password
	}
}

func (p *Platform) ToDTO() *types.Platform {
	return &types.Platform{
		ID:          p.ID,
		Type:        p.Type,
		Name:        p.Name,
		Description: p.Description.String,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: p.Username,
				Password: p.Password,
			},
		},
	}
}

func (p *Platform) FromDTO(platform *types.Platform) {
	*p = Platform{
		ID:          platform.ID,
		Type:        platform.Type,
		Name:        platform.Name,
		CreatedAt:   platform.CreatedAt,
		Description: sql.NullString{String: platform.Description},
		UpdatedAt:   platform.UpdatedAt,
	}

	if platform.Description != "" {
		p.Description.Valid = true
	}
	if platform.Credentials != nil && platform.Credentials.Basic != nil {
		p.Username = platform.Credentials.Basic.Username
		p.Password = platform.Credentials.Basic.Password
	}
}

func (so *ServiceOffering) ToDTO() *types.ServiceOffering {
	return &types.ServiceOffering{
		ID:                   so.ID,
		Name:                 so.Name,
		Description:          so.Description,
		CreatedAt:            so.CreatedAt,
		UpdatedAt:            so.UpdatedAt,
		Bindable:             so.Bindable,
		InstancesRetrievable: so.InstancesRetrievable,
		BindingsRetrievable:  so.BindingsRetrievable,
		PlanUpdatable:        so.PlanUpdatable,
		CatalogID:            so.CatalogID,
		CatalogName:          so.CatalogName,
		Tags:                 json.RawMessage(so.Tags),
		Requires:             json.RawMessage(so.Requires),
		Metadata:             json.RawMessage(so.Metadata),
		BrokerID:             so.BrokerID,
	}
}

func (so *ServiceOffering) FromDTO(offering *types.ServiceOffering) {
	*so = ServiceOffering{
		ID:                   offering.ID,
		Name:                 offering.Name,
		Description:          offering.Description,
		CreatedAt:            offering.CreatedAt,
		UpdatedAt:            offering.UpdatedAt,
		Bindable:             offering.Bindable,
		InstancesRetrievable: offering.InstancesRetrievable,
		BindingsRetrievable:  offering.BindingsRetrievable,
		PlanUpdatable:        offering.PlanUpdatable,
		CatalogID:            offering.CatalogID,
		CatalogName:          offering.CatalogName,
		Tags:                 sqlxtypes.JSONText(offering.Tags),
		Requires:             sqlxtypes.JSONText(offering.Requires),
		Metadata:             sqlxtypes.JSONText(offering.Metadata),
		BrokerID:             offering.BrokerID,
	}
}

func (sp *ServicePlan) ToDTO() *types.ServicePlan {
	return &types.ServicePlan{
		ID:                sp.ID,
		Name:              sp.Name,
		Description:       sp.Description,
		CreatedAt:         sp.CreatedAt,
		UpdatedAt:         sp.UpdatedAt,
		CatalogID:         sp.CatalogID,
		CatalogName:       sp.CatalogName,
		Free:              sp.Free,
		Bindable:          sp.Bindable,
		PlanUpdatable:     sp.PlanUpdatable,
		Metadata:          json.RawMessage(sp.Metadata),
		Schemas:           json.RawMessage(sp.Schemas),
		ServiceOfferingID: sp.ServiceOfferingID,
	}
}

func (sp *ServicePlan) FromDTO(plan *types.ServicePlan) {
	*sp = ServicePlan{
		ID:                plan.ID,
		Name:              plan.Name,
		Description:       plan.Description,
		CreatedAt:         plan.CreatedAt,
		UpdatedAt:         plan.UpdatedAt,
		Free:              plan.Free,
		Bindable:          plan.Bindable,
		PlanUpdatable:     plan.PlanUpdatable,
		CatalogID:         plan.CatalogID,
		CatalogName:       plan.CatalogName,
		Metadata:          sqlxtypes.JSONText(plan.Metadata),
		Schemas:           sqlxtypes.JSONText(plan.Schemas),
		ServiceOfferingID: plan.ServiceOfferingID,
	}
}

func (v *Visibility) ToDTO() *types.Visibility {
	return &types.Visibility{
		ID:            v.ID,
		PlatformID:    v.PlatformID.String,
		ServicePlanID: v.ServicePlanID,
		CreatedAt:     v.CreatedAt,
		UpdatedAt:     v.UpdatedAt,
	}
}

func (v *Visibility) FromDTO(visibility *types.Visibility) {
	*v = Visibility{
		ID:            visibility.ID,
		PlatformID:    sql.NullString{String: visibility.PlatformID},
		ServicePlanID: visibility.ServicePlanID,
		CreatedAt:     visibility.CreatedAt,
		UpdatedAt:     visibility.UpdatedAt,
	}
	// API cannot send nulls right now and storage cannot store empty string for this column as it is FK
	if visibility.PlatformID != "" {
		v.PlatformID.Valid = true
	}
}
