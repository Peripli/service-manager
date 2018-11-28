/*
 * Copyright 2018 The Service Manager Authors
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

import (
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type storageState struct {
	lastCheckTime        time.Time
	mutex                *sync.RWMutex
	db                   *sqlx.DB
	storageCheckInterval time.Duration
}

func (s *storageState) Get() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if time.Since(s.lastCheckTime) < s.storageCheckInterval {
		return nil
	}

	m := new(int64)
	return s.db.Get(m, "SELECT 1")
}
