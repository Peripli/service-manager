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

package types

import "time"

type Base struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Labels         Labels    `json:"labels,omitempty"`
	PagingSequence int64     `json:"-"`
	Ready          bool      `json:"ready"`

	LastOperation *Operation `json:"last_operation,omitempty"`
}

func (b *Base) SetID(id string) {
	b.ID = id
}

func (b *Base) GetID() string {
	return b.ID
}

func (b *Base) SetCreatedAt(time time.Time) {
	b.CreatedAt = time
}

func (b *Base) GetCreatedAt() time.Time {
	return b.CreatedAt
}

func (b *Base) SetUpdatedAt(time time.Time) {
	b.UpdatedAt = time
}

func (b *Base) GetUpdatedAt() time.Time {
	return b.UpdatedAt
}

func (b *Base) SetLabels(labels Labels) {
	b.Labels = labels
}

func (b *Base) GetLabels() Labels {
	return b.Labels
}

func (b *Base) ExtractTenantLabelValue(tenantLabelKey string) []string {
	tenant, ok := b.Labels[tenantLabelKey]
	if !ok {
		return nil
	}
	return tenant
}

func (b *Base) GetPagingSequence() int64 {
	return b.PagingSequence
}

func (b *Base) SetReady(ready bool) {
	b.Ready = ready
}

func (b *Base) GetReady() bool {
	return b.Ready
}

func (b *Base) SetLastOperation(op *Operation) {
	b.LastOperation = op
}

func (b *Base) GetLastOperation() *Operation {
	return b.LastOperation
}
