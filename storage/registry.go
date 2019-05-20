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
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"
)

func InitializeWithSafeTermination(ctx context.Context, s Storage, options *Settings, wg *sync.WaitGroup) error {
	if s == nil || options == nil {
		return fmt.Errorf("storage and storage settings cannot be nil")
	}

	if securityStorage, isSecured := s.(Secured); isSecured {
		if err := initializeSecureStorage(ctx, securityStorage); err != nil {
			panic(fmt.Sprintf("error initialzing secure storage: %v", err))
		}
	}

	util.StartInWaitGroupWithContext(ctx, func(c context.Context) {
		<-c.Done()
		log.C(c).Debug("Context cancelled. Closing storage...")
		if err := s.Close(); err != nil {
			log.D().Error(err)
		}
	}, wg)

	if err := s.Open(options); err != nil {
		return fmt.Errorf("error opening storage: %s", err)
	}

	return nil
}

func initializeSecureStorage(ctx context.Context, secureStorage Secured) error {
	ctx, cancelFunc := context.WithTimeout(ctx, 2*time.Second)
	defer cancelFunc()
	if err := secureStorage.Lock(ctx); err != nil {
		return err
	}

	encryptionKey, err := secureStorage.GetEncryptionKey(ctx)
	if err != nil {
		return err
	}
	if len(encryptionKey) == 0 {
		logger := log.C(ctx)
		logger.Info("No encryption key is present. Generating new one...")
		newEncryptionKey := make([]byte, 32)
		if _, err = rand.Read(newEncryptionKey); err != nil {
			return fmt.Errorf("could not generate encryption key: %v", err)
		}

		if err = secureStorage.SetEncryptionKey(ctx, newEncryptionKey); err != nil {
			return err
		}
		logger.Info("Successfully generated new encryption key")
	}
	return secureStorage.Unlock(ctx)
}
