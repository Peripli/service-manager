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
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

// BrokerPlatformCredential entity
//go:generate smgen storage BrokerPlatformCredential github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types
type BrokerPlatformCredential struct {
	BaseEntity

	Username        string `db:"username"`
	PasswordHash    string `db:"password_hash"`
	OldUsername     string `db:"old_username"`
	OldPasswordHash string `db:"old_password_hash"`

	PlatformID string `db:"platform_id"`
	BrokerID   string `db:"broker_id"`

	Integrity []byte `db:"integrity"`

	Active bool `db:"active"`
}

func (bpc *BrokerPlatformCredential) ToObject() (types.Object, error) {
	return &types.BrokerPlatformCredential{
		Base: types.Base{
			ID:             bpc.ID,
			CreatedAt:      bpc.CreatedAt,
			UpdatedAt:      bpc.UpdatedAt,
			Labels:         map[string][]string{},
			PagingSequence: bpc.PagingSequence,
			Ready:          bpc.Ready,
		},
		Username:        bpc.Username,
		PasswordHash:    bpc.PasswordHash,
		OldUsername:     bpc.OldUsername,
		OldPasswordHash: bpc.OldPasswordHash,
		PlatformID:      bpc.PlatformID,
		BrokerID:        bpc.BrokerID,
		Integrity:       bpc.Integrity,
		Active:          bpc.Active,
	}, nil
}

func (*BrokerPlatformCredential) FromObject(object types.Object) (storage.Entity, error) {
	brokerPlatformCredential, ok := object.(*types.BrokerPlatformCredential)
	if !ok {
		return nil, fmt.Errorf("object is not of type BrokerPlatformCredential")
	}

	bpc := &BrokerPlatformCredential{
		BaseEntity: BaseEntity{
			ID:             brokerPlatformCredential.ID,
			CreatedAt:      brokerPlatformCredential.CreatedAt,
			UpdatedAt:      brokerPlatformCredential.UpdatedAt,
			PagingSequence: brokerPlatformCredential.PagingSequence,
			Ready:          brokerPlatformCredential.Ready,
		},
		Username:        brokerPlatformCredential.Username,
		PasswordHash:    brokerPlatformCredential.PasswordHash,
		OldUsername:     brokerPlatformCredential.OldUsername,
		OldPasswordHash: brokerPlatformCredential.OldPasswordHash,
		PlatformID:      brokerPlatformCredential.PlatformID,
		BrokerID:        brokerPlatformCredential.BrokerID,
		Integrity:       brokerPlatformCredential.Integrity,
		Active:          brokerPlatformCredential.Active,
	}

	return bpc, nil
}
