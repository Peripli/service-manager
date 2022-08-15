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
	"fmt"

	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

// Notification entity
//go:generate smgen storage notification github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types:Notification
type Notification struct {
	BaseEntity
	Resource      string             `db:"resource"`
	Type          string             `db:"type"`
	PlatformID    sql.NullString     `db:"platform_id"`
	Revision      int64              `db:"revision,auto_increment"`
	Payload       sqlxtypes.JSONText `db:"payload"`
	CorrelationID sql.NullString     `db:"correlation_id"`
}

func (n *Notification) ToObject() (types.Object, error) {
	return &types.Notification{
		Base: types.Base{
			ID:        n.ID,
			CreatedAt: n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
			Labels:    map[string][]string{},
			Ready:     n.Ready,
		},
		Resource:      types.ObjectType(n.Resource),
		Type:          types.NotificationOperation(n.Type),
		PlatformID:    n.PlatformID.String,
		Revision:      n.Revision,
		Payload:       getJSONRawMessage(n.Payload),
		CorrelationID: n.CorrelationID.String,
	}, nil
}

func (*Notification) FromObject(object types.Object) (storage.Entity, error) {
	notification, ok := object.(*types.Notification)
	if !ok {
		return nil, fmt.Errorf("object is not of type Notification")
	}

	n := &Notification{
		BaseEntity: BaseEntity{
			ID:        notification.ID,
			CreatedAt: notification.CreatedAt,
			UpdatedAt: notification.UpdatedAt,
			Ready:     notification.Ready,
		},
		Resource:      string(notification.Resource),
		Type:          string(notification.Type),
		PlatformID:    toNullString(notification.PlatformID),
		Revision:      notification.Revision, // when creating new Notification, Revision will be set by DB
		Payload:       getJSONText(notification.Payload),
		CorrelationID: toNullString(notification.CorrelationID),
	}
	return n, nil
}
