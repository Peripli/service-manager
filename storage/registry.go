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

package storage

import (
	"context"
	"sync"

	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	mux      sync.RWMutex
	storages = make(map[string]Storage)
)

// Register adds a storage with the given name
func Register(name string, storage Storage) {
	mux.RLock()
	defer mux.RUnlock()
	if storage == nil {
		logrus.Panicln("storage: Register storage is nil")
	}
	if _, dup := storages[name]; dup {
		logrus.Panicf("storage: Register called twice for storage with name %s", name)
	}
	storages[name] = storage
}

// Use specifies the storage for the given name
// Returns the storage ready to be used and an error if one occurred during initialization
// Upon context.Done signal the storage will be closed
func Use(ctx context.Context, name string, uri string, encryptionKey []byte) (Storage, error) {
	mux.Lock()
	defer mux.Unlock()
	storage, exists := storages[name]
	if !exists {
		return nil, fmt.Errorf("error locating storage with name %s", name)
	}
	if err := storage.Open(uri, encryptionKey); err != nil {
		return nil, fmt.Errorf("error opening storage: %s", err)
	}
	storages[name] = storage
	go awaitTermination(ctx, storage)
	return storage, nil
}

func awaitTermination(ctx context.Context, storage Storage) {
	<-ctx.Done()
	logrus.Debug("Context cancelled. Closing storage...")
	if err := storage.Close(); err != nil {
		logrus.Error(err)
	}
}
