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
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

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

// Use specifies the storage for the given name
func Use(ctx context.Context, name string, storage Storage) {
	mux.Lock()
	defer mux.Unlock()
	if _, exists := storages[name]; exists {
		logrus.Panic("A provider with this name has already been registered")
	}
	if err := storage.Open(); err != nil {
		logrus.Panic(err)
	}
	storages[name] = storage
	go awaitTermination(ctx, storage)
}

func awaitTermination(ctx context.Context, storage Storage) {
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		logrus.Debug("Received sigterm. Closing storage...")
		closeStorage(storage)
	case <-ctx.Done():
		logrus.Debug("Context cancelled. Closing storage...")
		closeStorage(storage)
	}
}

func closeStorage(storage Storage) {
	if err := storage.Close(); err != nil {
		logrus.Panic(err)
	}
}
