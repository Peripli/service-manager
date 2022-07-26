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

import "github.com/Peripli/service-manager/pkg/types"

// ServiceBinding entity
//go:generate smgen storage ServiceBinding github.com/Peripli/service-manager/pkg/types
type BlockedClient struct {
	BaseEntity
	ClientID       string   `db:"client_id"`
	SubaccountID   string   `db:"subaccount_id"`
	BlockedMethods []string `db:"blocked_methods"`
}

func (bc *BlockedClient) ToObject() (types.Object, error) {
	return &types.BlockedClient{
		Base: types.Base{
			ID:             bc.ID,
			CreatedAt:      bc.CreatedAt,
			UpdatedAt:      bc.UpdatedAt,
			Labels:         map[string][]string{},
			PagingSequence: bc.PagingSequence,
			Ready:          bc.Ready,
		},
		ClientID:       bc.ClientID,
		SubaccountID:   bc.SubaccountID,
		BlockedMethods: bc.BlockedMethods,
	}, nil
}
