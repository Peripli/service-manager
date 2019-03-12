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
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (e *Base) SetID(id string) {
	e.ID = id
}

func (e *Base) GetID() string {
	return e.ID
}

func (e *Base) SetCreatedAt(time time.Time) {
	e.CreatedAt = time
}

func (e *Base) GetCreatedAt() time.Time {
	return e.CreatedAt
}

func (e *Base) SetUpdatedAt(time time.Time) {
	e.UpdatedAt = time
}

func (e *Base) GetUpdatedAt() time.Time {
	return e.UpdatedAt
}

func (e *Base) SupportsLabels() bool {
	return false
}

func (e *Base) SetLabels(labels Labels) {
	return
}

func (e *Base) GetLabels() Labels {
	return Labels{}
}

type BaseLabelled struct {
	Base
	Labels Labels `json:"labels,omitempty"`
}

func (e *BaseLabelled) SetLabels(labels Labels) {
	e.Labels = labels
	return
}

func (e *BaseLabelled) GetLabels() Labels {
	return e.Labels
}

func (e *BaseLabelled) SupportsLabels() bool {
	return true
}
