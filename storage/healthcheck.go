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

package storage

import (
	"github.com/InVisionApp/go-health/checkers"
	"github.com/Peripli/service-manager/pkg/health"
	"time"
)

// HealthIndicator returns a new indicator for the storage
type HealthIndicator struct {
	*checkers.SQL

	settings *health.IndicatorSettings
}

// Name returns the name of the storage component
func (i *HealthIndicator) Name() string {
	return "storage"
}

func (i *HealthIndicator) Configure(settings *health.IndicatorSettings) {
	i.settings = settings
}

func (i *HealthIndicator) Interval() time.Duration {
	return i.settings.Interval
}

func (i *HealthIndicator) FailuresTreshold() int64 {
	return i.settings.FailuresTreshold
}

func (i *HealthIndicator) Fatal() bool {
	return i.settings.Fatal
}

func NewStorageHealthIndicator(pingFunc PingFunc) (health.Indicator, error) {
	sqlConfig := &checkers.SQLConfig{
		Pinger: pingFunc,
	}
	sqlChecker, err := checkers.NewSQL(sqlConfig)
	if err != nil {
		return nil, err
	}

	indicator := &HealthIndicator{
		SQL: sqlChecker,
	}

	return indicator, nil

}
