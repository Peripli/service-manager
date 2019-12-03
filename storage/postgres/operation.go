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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

// Operation entity
//go:generate smgen storage operation github.com/Peripli/service-manager/pkg/types:Operation
type Operation struct {
	BaseEntity
	Description   sql.NullString     `db:"description"`
	Type          string             `db:"type"`
	State         string             `db:"state"`
	ResourceID    string             `db:"resource_id"`
	ResourceType  string             `db:"resource_type"`
	Errors        sqlxtypes.JSONText `db:"errors"`
	CorrelationID sql.NullString     `db:"correlation_id"`
	ExternalID    sql.NullString     `db:"external_id"`
}

func (o *Operation) ToObject() types.Object {
	return &types.Operation{
		Base: types.Base{
			ID:             o.ID,
			CreatedAt:      o.CreatedAt,
			UpdatedAt:      o.UpdatedAt,
			PagingSequence: o.PagingSequence,
		},
		Description:   o.Description.String,
		Type:          types.OperationCategory(o.Type),
		State:         types.OperationState(o.State),
		ResourceID:    o.ResourceID,
		ResourceType:  o.ResourceType,
		Errors:        getJSONRawMessage(o.Errors),
		CorrelationID: o.CorrelationID.String,
		ExternalID:    o.ExternalID.String,
	}
}

func (*Operation) FromObject(object types.Object) (storage.Entity, bool) {
	operation, ok := object.(*types.Operation)
	if !ok {
		return nil, false
	}

	o := &Operation{
		BaseEntity: BaseEntity{
			ID:             operation.ID,
			CreatedAt:      operation.CreatedAt,
			UpdatedAt:      operation.UpdatedAt,
			PagingSequence: operation.PagingSequence,
		},
		Description:   toNullString(operation.Description),
		Type:          string(operation.Type),
		State:         string(operation.State),
		ResourceID:    operation.ResourceID,
		ResourceType:  operation.ResourceType,
		Errors:        getJSONText(operation.Errors),
		CorrelationID: toNullString(operation.CorrelationID),
		ExternalID:    toNullString(operation.ExternalID),
	}
	return o, true
}
