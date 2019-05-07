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

// Notification entity
//go:generate smgen storage notification github.com/Peripli/service-manager/pkg/types:Notification
type Notification struct {
	BaseEntity
	Resource   string             `db:"resource"`
	Type       string             `db:"type"`
	PlatformID sql.NullString     `db:"platform_id"`
	Revision   int64              `db:"revision,auto_increment"`
	Payload    sqlxtypes.JSONText `db:"payload"`
}

func (n *Notification) ToObject() types.Object {
	return &types.Notification{
		Base: types.Base{
			ID:        n.ID,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
			Labels:    map[string][]string{},
		},
		Resource:   types.ObjectType(n.Resource),
		Type:       types.OperationType(n.Type),
		PlatformID: n.PlatformID.String,
		Revision:   n.Revision,
		Payload:    getJSONRawMessage(n.Payload),
	}
}

func (*Notification) FromObject(object types.Object) (storage.Entity, bool) {
	notification, ok := object.(*types.Notification)
	if !ok {
		return nil, false
	}

	platformID := sql.NullString{
		String: notification.PlatformID,
		Valid:  notification.PlatformID != "",
	}

	n := &Notification{
		BaseEntity: BaseEntity{
			ID:        notification.ID,
			CreatedAt: notification.CreatedAt,
			UpdatedAt: notification.UpdatedAt,
		},
		Resource:   string(notification.Resource),
		Type:       string(notification.Type),
		PlatformID: platformID,
		Revision:   notification.Revision, // when creating new Notification, Revision will be set by DB
		Payload:    getJSONText(notification.Payload),
	}
	return n, true
}
