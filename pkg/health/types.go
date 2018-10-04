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

package health

type Status string

const (
	StatusUp      Status = "UP"
	StatusDown           = "DOWN"
	StatusUnknown        = "UNKNOWN"
)

type Health struct {
	Status  Status                 `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func NewHealth() *Health {
	return &Health{
		Status:  StatusUnknown,
		Details: make(map[string]interface{}),
	}
}

func (h *Health) WithStatus(status Status) *Health {
	h.Status = status
	return h
}

func (h *Health) WithError(err error) *Health {
	return h.WithDetail("error", err)
}

func (h *Health) WithDetail(key string, val interface{}) *Health {
	h.Details[key] = val
	return h
}

func (h *Health) Up() *Health {
	h.Status = StatusUp
	return h
}

func (h *Health) Down() *Health {
	h.Status = StatusDown
	return h
}

func (h *Health) WithDetails(details map[string]interface{}) *Health {
	h.Details = details
	return h
}

type Indicator interface {
	Name() string
	Health() *Health
}

type Aggregator interface {
	Aggregate(healths map[string]*Health) *Health
}

type Provider interface {
	AddHealthIndicator(indicator Indicator)
	HealthIndicators() []Indicator
}
