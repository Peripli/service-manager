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

package storage

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

var (
	mux      sync.RWMutex
	storages = make(map[string]Storage)
)

// Get returns a single storage
// Panics if more than one storage is configured. Use GetByName in such cases
func Get() Storage {
	mux.RLock()
	defer mux.RUnlock()
	storagesCount := len(storages)
	if storagesCount != 1 {
		logrus.Panicf("Requested exactly one storage but %d storages are configured", storagesCount)
	}
	var registeredStorage Storage
	for _, v := range storages {
		registeredStorage = v
		break
	}
	return registeredStorage
}

// GetByName returns a storage with this name and boolean indicating whether it exists
func GetByName(name string) (Storage, bool) {
	mux.RLock()
	defer mux.RUnlock()
	providedStorage, exists := storages[name]
	return providedStorage, exists
}

// Register registers a storage with the specified name
func Register(name string, storage Storage) {
	if storage == nil {
		logrus.Panic("Cannot register nil storage")
	}
	mux.Lock()
	defer mux.Unlock()
	if _, exists := storages[name]; exists {
		logrus.Panic("A storage with this name has already been registered")
	}
	storages[name] = storage
}
