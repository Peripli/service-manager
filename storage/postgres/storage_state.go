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

// Package postgres implements the Service Manager storage interfaces for Postgresql Storage
package postgres

import (
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type storageState struct {
	storageError         error
	lastCheck            time.Time
	mutex                *sync.RWMutex
	db                   *sqlx.DB
	storageCheckInterval time.Duration
}

// Get returns error if the db connectivity is down and nil otherwise
func (state *storageState) Get() error {
	if cacheIsValid, storageError := state.getCashed(); cacheIsValid {
		return storageError
	}
	return state.checkDB()
}

func (state *storageState) cachedStateIsValid() bool {
	return time.Now().Sub(state.lastCheck) < state.storageCheckInterval
}

func (state *storageState) getCashed() (cacheIsValid bool, storageError error) {
	state.mutex.RLock()
	defer state.mutex.RUnlock()
	if state.cachedStateIsValid() {
		return true, state.storageError
	}
	return false, nil
}

func (state *storageState) checkDB() error {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	// check if someone hasn't updated the cached state already
	if state.cachedStateIsValid() {
		return state.storageError
	}
	if _, err := state.db.Query("SELECT 1"); err != nil {
		state.storageError = err
	} else {
		state.storageError = nil
	}
	state.lastCheck = time.Now()
	return state.storageError
}
