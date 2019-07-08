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

import (
	"fmt"
	"github.com/InVisionApp/go-health"
	"github.com/Peripli/service-manager/pkg/log"
	"time"
)

// Settings type to be loaded from the environment
type Settings struct {
	IndicatorsSettings map[string]*IndicatorSettings `mapstructure:"indicators,omitempty"`
}

// DefaultSettings returns default values for health settings
func DefaultSettings() *Settings {
	emptySettings := make(map[string]*IndicatorSettings)
	return &Settings{
		IndicatorsSettings: emptySettings,
	}
}

// Validate validates health settings
func (s *Settings) Validate() error {
	for _, v := range s.IndicatorsSettings {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// IndicatorSettings type to be loaded from the environment
type IndicatorSettings struct {
	Fatal            bool          `mapstructure:"fatal" description:"if the indicator affects the overall status "`
	FailuresTreshold int64         `mapstructure:"failures_treshold" description:"maximum failures in a row until component is considered down"`
	Interval         time.Duration `mapstructure:"interval" description:"time between health checks of components"`
}

// DefaultIndicatorSettings returns default values for indicator settings
func DefaultIndicatorSettings() *IndicatorSettings {
	return &IndicatorSettings{
		Fatal:            true,
		FailuresTreshold: 3,
		Interval:         60,
	}
}

// Validate validates indicator settings
func (is *IndicatorSettings) Validate() error {
	if is.FailuresTreshold <= 0 {
		return fmt.Errorf("validate Settings: FailuresTreshold must be > 0")
	}
	if is.Interval < 30*time.Second {
		return fmt.Errorf("validate Settings: Minimum interval is 30 seconds")
	}
	return nil
}

// Status represents the overall health status of a component
type Status string

const (
	// StatusUp indicates that the checked component is up and running
	StatusUp Status = "UP"
	// StatusDown indicates the the checked component has an issue and is unavailable
	StatusDown Status = "DOWN"
	// StatusUnknown indicates that the health of the checked component cannot be determined
	StatusUnknown Status = "UNKNOWN"
)

type StatusListener struct{}

func (sl *StatusListener) HealthCheckFailed(state *health.State) {
	log.D().Errorf("Health check for %v failed with: %v", state.Name, state.Err)
}

func (sl *StatusListener) HealthCheckRecovered(state *health.State, numberOfFailures int64, unavailableDuration float64) {
	log.D().Infof("Health check for %v recovered after %v failures and was unavailable for %v seconds roughly", state.Name, numberOfFailures, unavailableDuration)
}

// Health contains information about the health of a component.
type Health struct {
	Status  Status                 `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// New returns a new Health with an unknown status an empty details.
func New() *Health {
	return &Health{
		Status:  StatusUnknown,
		Details: make(map[string]interface{}),
	}
}

// WithStatus sets the status of the health
func (h *Health) WithStatus(status Status) *Health {
	h.Status = status
	return h
}

// WithError sets the status of the health to DOWN and adds an error detail
func (h *Health) WithError(err error) *Health {
	h.Status = StatusDown
	return h.WithDetail("error", err)
}

// WithDetail adds a detail to the health
func (h *Health) WithDetail(key string, val interface{}) *Health {
	h.Details[key] = val
	return h
}

// WithDetails adds the given details to the health
func (h *Health) WithDetails(details map[string]interface{}) *Health {
	for k, v := range details {
		h.Details[k] = v
	}
	return h
}

// Indicator is an interface to provide the health of a component
//go:generate counterfeiter . Indicator
type Indicator interface {
	// Name returns the name of the component
	Name() string

	// Interval returns settings of the indicator
	Interval() time.Duration

	// FailuresTreshold returns the maximum failures in a row until component is considered down
	FailuresTreshold() int64

	// Fatal returns if the health indicator is fatal for the overall status
	Fatal() bool

	// Status returns the health information of the component
	Status() (interface{}, error)
}

// ConfigurableIndicator is an interface to provide configurable health of a component
//go:generate counterfeiter . ConfigurableIndicator
type ConfigurableIndicator interface {
	Indicator

	// Configure configures the indicator with given settings
	Configure(*IndicatorSettings)
}

// NewDefaultRegistry returns a default health registry with a single ping indicator
func NewDefaultRegistry() *Registry {
	emptySettings := make(map[string]*IndicatorSettings)
	return &Registry{
		HealthIndicators: []Indicator{&pingIndicator{}},
		HealthSettings:   emptySettings,
	}
}

// Registry is an interface to store and fetch health indicators
type Registry struct {
	// HealthIndicators are the currently registered health indicators
	HealthIndicators []Indicator

	// Indicator Settings of the registry
	HealthSettings map[string]*IndicatorSettings
}

// ConfigureIndicators configures registry's indicators with provided settings
func (r *Registry) ConfigureIndicators() {
	for _, hi := range r.HealthIndicators {
		if indicator, ok := hi.(ConfigurableIndicator); ok {
			if settings, ok := r.HealthSettings[indicator.Name()]; ok {
				indicator.Configure(settings)
			} else {
				indicator.Configure(DefaultIndicatorSettings())
			}
		}
	}
}
