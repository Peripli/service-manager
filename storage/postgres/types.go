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
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/types"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

const (
	// platformTable db table name for platforms
	platformTable = "platforms"

	// brokerTable db table name for brokers
	brokerTable = "brokers"

	// brokerLabelsTable db table for broker labels
	brokerLabelsTable = "DeleteInterceptor"

	// serviceOfferingTable db table for service offerings
	serviceOfferingTable = "service_offerings"

	// servicePlanTable db table for service plans
	servicePlanTable = "service_plans"

	// visibilityTable db table for visibilities
	visibilityTable = "visibilities"

	// visibilityLabelsTable db table for visibilities table
	visibilityLabelsTable = "visibility_labels"
)

// Safe represents a secret entity
type Safe struct {
	Secret    []byte    `db:"secret"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

//func init() {
//	RegisterEntity(types.ServiceOfferingType, &ServiceOffering{})
//}

type ServiceOffering struct {
	*BaseEntity
	Name        string `db:"name"`
	Description string `db:"description"`

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

func (so *ServiceOffering) LabelEntity() LabelEntity {
	return nil
}

func (so *ServiceOffering) GetType() types.ObjectType {
	panic("implement me")
}

func (so *ServiceOffering) SupportsLabels() bool {
	panic("implement me")
}

func (so *ServiceOffering) GetLabels() types.Labels {
	panic("implement me")
}

func (so *ServiceOffering) EmptyList() types.ObjectList {
	panic("implement me")
}

func (so *ServiceOffering) WithLabels(labels types.Labels) types.Object {
	panic("implement me")
}

func (so *ServiceOffering) GetID() string {
	return so.ID
}

func (so *ServiceOffering) TableName() string {
	return serviceOfferingTable
}

func (so *ServiceOffering) PrimaryColumn() string {
	return "id"
}

func (so *ServiceOffering) Empty() Entity {
	return &ServiceOffering{}
}

func (so *ServiceOffering) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	target := []*ServiceOffering{}
	for rows.Next() {
		var item ServiceOffering
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		target = append(target, &item)
	}
	result := &types.ServiceOfferings{}
	for _, v := range target {
		result.Add(v.ToObject())
	}
	return result, nil
}

func (so *ServiceOffering) FromObject(object types.Object) Entity {
	offering := object.(*types.ServiceOffering)
	result := &ServiceOffering{
		BaseEntity: &BaseEntity{
			ID:        offering.ID,
			CreatedAt: offering.CreatedAt,
			UpdatedAt: offering.UpdatedAt,
		},
		Name:                 offering.Name,
		Description:          offering.Description,
		Bindable:             offering.Bindable,
		InstancesRetrievable: offering.InstancesRetrievable,
		BindingsRetrievable:  offering.BindingsRetrievable,
		PlanUpdatable:        offering.PlanUpdatable,
		CatalogID:            offering.CatalogID,
		CatalogName:          offering.CatalogName,
		Tags:                 getJSONText(offering.Tags),
		Requires:             getJSONText(offering.Requires),
		Metadata:             getJSONText(offering.Metadata),
		BrokerID:             offering.BrokerID,
	}
	return result
}

func (so *ServiceOffering) ToObject() types.Object {
	return &types.ServiceOffering{
		Base: &types.Base{
			ID:        so.ID,
			CreatedAt: so.CreatedAt,
			UpdatedAt: so.UpdatedAt,
		},
		Name:                 so.Name,
		Description:          so.Description,
		Bindable:             so.Bindable,
		InstancesRetrievable: so.InstancesRetrievable,
		BindingsRetrievable:  so.BindingsRetrievable,
		PlanUpdatable:        so.PlanUpdatable,
		CatalogID:            so.CatalogID,
		CatalogName:          so.CatalogName,
		Tags:                 getJSONRawMessage(so.Tags),
		Requires:             getJSONRawMessage(so.Requires),
		Metadata:             getJSONRawMessage(so.Metadata),
		BrokerID:             so.BrokerID,
	}
}

func getJSONText(item json.RawMessage) sqlxtypes.JSONText {
	if len(item) == len("null") && string(item) == "null" {
		return sqlxtypes.JSONText("{}")
	}
	return sqlxtypes.JSONText(item)
}

func getJSONRawMessage(item sqlxtypes.JSONText) json.RawMessage {
	if len(item) <= len("null") {
		itemStr := string(item)
		if itemStr == "{}" || itemStr == "null" {
			return nil
		}
	}
	return json.RawMessage(item)
}
