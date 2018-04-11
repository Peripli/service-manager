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

// Argumentation for refactoring
//Use returning the storage seems more natural rather then doing .Get; .Get() introduces a hidden dependency as well)
//This way we kind of provide a wrapper over the sqlx.db by keeping the same flow of usage while keeping an abstraction for Storage possible
// Wrapping as follows
// - postgres impl of storage _ imports the driver, we _import the postgres storage's package
// Registering a storage "wraps" registering a driver and allows us to "use" the storage that wraps abstract opening a storage (in posgres it opens sqlx.db)
// The idea is to allow abstraction of Storage while keeping the sqlx.db flow of usage as close as possible
// uri is passed as argument as we don't want to obtain env variables in random places of the code as the config might not be provided as env vars)
// It's better to abstract away the environment loading and use it to build a configuration that is passed to the server. The server than takes care
// to split the configuration and pass segments to the components it initializes (to storage the server will pass uri)
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
