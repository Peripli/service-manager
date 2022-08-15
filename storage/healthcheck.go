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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/health"
)

// NewSQLHealthIndicator returns new health indicator for sql storage given a ping function
func NewSQLHealthIndicator(pingFunc PingFunc) (health.Indicator, error) {
	sqlConfig := &checkers.SQLConfig{
		Pinger: pingFunc,
	}
	sqlChecker, err := checkers.NewSQL(sqlConfig)
	if err != nil {
		return nil, err
	}

	indicator := &SQLHealthIndicator{
		SQL: sqlChecker,
	}

	return indicator, nil
}

// SQLHealthIndicator returns a new indicator for SQL storage
type SQLHealthIndicator struct {
	*checkers.SQL
}

// Name returns the name of the storage component
func (i *SQLHealthIndicator) Name() string {
	return health.StorageIndicatorName
}
