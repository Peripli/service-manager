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

package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

// Operation entity
//go:generate smgen storage operation github.com/Peripli/service-manager/pkg/types:Operation
type Operation struct {
	BaseEntity
	Description         sql.NullString     `db:"description"`
	Type                string             `db:"type"`
	State               string             `db:"state"`
	ResourceID          string             `db:"resource_id"`
	TransitiveResources sqlxtypes.JSONText `db:"transitive_resources"`
	ResourceType        string             `db:"resource_type"`
	PlatformID          string             `db:"platform_id"`
	Errors              sqlxtypes.JSONText `db:"errors"`
	CorrelationID       sql.NullString     `db:"correlation_id"`
	ExternalID          sql.NullString     `db:"external_id"`
	Reschedule          bool               `db:"reschedule"`
	RescheduleTimestamp time.Time          `db:"reschedule_timestamp"`
	DeletionScheduled   time.Time          `db:"deletion_scheduled"`
}

func (o *Operation) ToObject() (types.Object, error) {
	transitiveResources := make([]*types.RelatedType, 0)
	if o.TransitiveResources.String() != "" {
		if err := util.BytesToObject(getJSONRawMessage(o.TransitiveResources), &transitiveResources); err != nil {
			return nil, err
		}
	}

	return &types.Operation{
		Base: types.Base{
			ID:             o.ID,
			CreatedAt:      o.CreatedAt,
			UpdatedAt:      o.UpdatedAt,
			PagingSequence: o.PagingSequence,
			Ready:          o.Ready,
		},
		Description:         o.Description.String,
		Type:                types.OperationCategory(o.Type),
		State:               types.OperationState(o.State),
		ResourceID:          o.ResourceID,
		TransitiveResources: transitiveResources,
		ResourceType:        types.ObjectType(o.ResourceType),
		PlatformID:          o.PlatformID,
		Errors:              getJSONRawMessage(o.Errors),
		CorrelationID:       o.CorrelationID.String,
		ExternalID:          o.ExternalID.String,
		Reschedule:          o.Reschedule,
		RescheduleTimestamp: o.RescheduleTimestamp,
		DeletionScheduled:   o.DeletionScheduled,
	}, nil
}

func (*Operation) FromObject(object types.Object) (storage.Entity, error) {
	operation, ok := object.(*types.Operation)
	if !ok {
		return nil, fmt.Errorf("object is not of type Operation")
	}
	if operation.TransitiveResources == nil {
		operation.TransitiveResources = make([]*types.RelatedType, 0)
	}
	transitiveResourcesBytes, err := json.Marshal(operation.TransitiveResources)
	if err != nil {
		log.D().Errorf("Could not marshal transitive resources of operation: %s", err.Error())
		return nil, err
	}

	o := &Operation{
		BaseEntity: BaseEntity{
			ID:             operation.ID,
			CreatedAt:      operation.CreatedAt,
			UpdatedAt:      operation.UpdatedAt,
			PagingSequence: operation.PagingSequence,
			Ready:          operation.Ready,
		},
		Description:         toNullString(operation.Description),
		Type:                string(operation.Type),
		State:               string(operation.State),
		ResourceID:          operation.ResourceID,
		TransitiveResources: getJSONText(transitiveResourcesBytes),
		ResourceType:        operation.ResourceType.String(),
		PlatformID:          operation.PlatformID,
		Errors:              getJSONText(operation.Errors),
		CorrelationID:       toNullString(operation.CorrelationID),
		ExternalID:          toNullString(operation.ExternalID),
		Reschedule:          operation.Reschedule,
		RescheduleTimestamp: operation.RescheduleTimestamp,
		DeletionScheduled:   operation.DeletionScheduled,
	}
	return o, nil
}
