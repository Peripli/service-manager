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
	"sync"

	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	mux      sync.RWMutex
	storages = make(map[string]Storage)
)

func Register(name string, storage Storage) {
	mux.RLock()
	defer mux.RUnlock()
	if storage == nil {
		panic("storage: Register storage is nil")
	}
	if _, dup := storages[name]; dup {
		panic("storage: Register called twice for storage with name " + name)
	}
	storages[name] = storage
}

func Use(name string, uri string, ctx context.Context) (Storage, error) {
	mux.Lock()
	defer mux.Unlock()
	storage, exists := storages[name]
	if !exists {
		return nil, fmt.Errorf("Error locating storage with name %s", name)
	}
	if err := storage.Open(uri); err != nil {
		return nil, fmt.Errorf("Error opening storage: %s", err)
	}
	storages[name] = storage
	go awaitTermination(storage, ctx)
	return storage, nil
}

func awaitTermination(storage Storage, ctx context.Context) {
	select {
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
