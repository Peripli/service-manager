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
	"context"
	"fmt"
	"github.com/InVisionApp/go-health"
	h "github.com/InVisionApp/go-health"
	l "github.com/InVisionApp/go-logger/shims/logrus"
	"github.com/Peripli/service-manager/pkg/log"
	"time"
)

// Settings type to be loaded from the environment
type Settings struct {
	Indicators map[string]*IndicatorSettings `mapstructure:"indicators,omitempty"`
}

// DefaultSettings returns default values for health settings
func DefaultSettings() *Settings {
	emptySettings := make(map[string]*IndicatorSettings)
	return &Settings{
		Indicators: emptySettings,
	}
}

// Validate validates health settings
func (s *Settings) Validate() error {
	for _, v := range s.Indicators {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// IndicatorSettings type to be loaded from the environment
type IndicatorSettings struct {
	Fatal             bool          `mapstructure:"fatal" description:"if the indicator affects the overall status, if false not failures_threshold expected"`
	FailuresThreshold int64         `mapstructure:"failures_threshold" description:"number of failures in a row that will affect overall status"`
	Interval          time.Duration `mapstructure:"interval" description:"time between health checks of components"`
}

// DefaultIndicatorSettings returns default values for indicator settings
func DefaultIndicatorSettings() *IndicatorSettings {
	return &IndicatorSettings{
		Fatal:             true,
		FailuresThreshold: 3,
		Interval:          60 * time.Second,
	}
}

// Validate validates indicator settings
func (is *IndicatorSettings) Validate() error {
	if !is.Fatal && is.FailuresThreshold != 0 {
		return fmt.Errorf("validate Settings: FailuresThreshold not applicable for non-fatal indicators")
	}
	if is.Fatal && is.FailuresThreshold <= 0 {
		return fmt.Errorf("validate Settings: FailuresThreshold must be > 0 for fatal indicators")
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

	// Status returns the health information of the component
	Status() (interface{}, error)
}

// NewDefaultRegistry returns a default empty health registry
func NewDefaultRegistry() *Registry {
	return &Registry{
		HealthIndicators: make([]Indicator, 0),
	}
}

// Registry is a struct to store health indicators
type Registry struct {
	// HealthIndicators are the currently registered health indicators
	HealthIndicators []Indicator
}

// Configure creates new health using provided settings.
func Configure(ctx context.Context, indicators []Indicator, settings *Settings) (*h.Health, map[string]int64, error) {
	healthz := h.New()
	logger := log.C(ctx).Logger

	healthz.Logger = l.New(logger)
	healthz.StatusListener = &StatusListener{}

	thresholds := make(map[string]int64)

	for _, indicator := range indicators {
		s, ok := settings.Indicators[indicator.Name()]
		if !ok {
			s = DefaultIndicatorSettings()
		}
		if err := healthz.AddCheck(&h.Config{
			Name:     indicator.Name(),
			Checker:  indicator,
			Interval: s.Interval,
			Fatal:    s.Fatal,
		}); err != nil {
			return nil, nil, err
		}
		thresholds[indicator.Name()] = s.FailuresThreshold
	}
	return healthz, thresholds, nil
}
