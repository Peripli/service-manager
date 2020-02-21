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
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
)

// BrokerPlatformCredential entity
//go:generate smgen storage BrokerPlatformCredential github.com/Peripli/service-manager/pkg/types
type BrokerPlatformCredential struct {
	BaseEntity

	Username    string `db:"username"`
	Password    string `db:"password"`
	OldUsername string `db:"old_username"`
	OldPassword string `db:"old_password"`

	PlatformID string `db:"platform_id"`
	BrokerID   string `db:"broker_id"`
}

func (bpc *BrokerPlatformCredential) ToObject() types.Object {
	return &types.BrokerPlatformCredential{
		Base: types.Base{
			ID:             bpc.ID,
			CreatedAt:      bpc.CreatedAt,
			UpdatedAt:      bpc.UpdatedAt,
			Labels:         map[string][]string{},
			PagingSequence: bpc.PagingSequence,
			Ready:          bpc.Ready,
		},
		Username:    bpc.Username,
		Password:    bpc.Password,
		OldUsername: bpc.OldUsername,
		OldPassword: bpc.OldPassword,
		PlatformID:  bpc.PlatformID,
		BrokerID:    bpc.BrokerID,
	}
}

func (*BrokerPlatformCredential) FromObject(object types.Object) (storage.Entity, bool) {
	brokerPlatformCredential, ok := object.(*types.BrokerPlatformCredential)
	if !ok {
		return nil, false
	}

	bpc := &BrokerPlatformCredential{
		BaseEntity: BaseEntity{
			ID:             brokerPlatformCredential.ID,
			CreatedAt:      brokerPlatformCredential.CreatedAt,
			UpdatedAt:      brokerPlatformCredential.UpdatedAt,
			PagingSequence: brokerPlatformCredential.PagingSequence,
			Ready:          brokerPlatformCredential.Ready,
		},
		Username:    brokerPlatformCredential.Username,
		Password:    brokerPlatformCredential.Password,
		OldUsername: brokerPlatformCredential.OldUsername,
		OldPassword: brokerPlatformCredential.OldUsername,
		PlatformID:  brokerPlatformCredential.PlatformID,
		BrokerID:    brokerPlatformCredential.BrokerID,
	}

	return bpc, true
}
