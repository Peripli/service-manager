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

package sm

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
)

// generates a new encryption key only if the node is a leader. Otherwise waits for the leader to generate one
func initializeSecureStorage(secureStorage storage.Security, isLeader bool) error {
	keyFetcher := secureStorage.Fetcher()
	encryptionKey, err := keyFetcher.GetEncryptionKey()
	if err != nil {
		return err
	}
	if len(encryptionKey) == 0 {
		if isLeader {
			return generateEncryptionKey(secureStorage.Setter())
		}
		return waitForLeader(keyFetcher)
	}
	return nil
}

func generateEncryptionKey(keySetter security.KeySetter) error {
	logrus.Debug("Leader: No encryption key is present. Generating new one...")
	newEncryptionKey := make([]byte, 32)
	if _, err := rand.Read(newEncryptionKey); err != nil {
		return fmt.Errorf("Could not generate encryption key: %v", err)
	}
	if err := keySetter.SetEncryptionKey(newEncryptionKey); err != nil {
		return err
	}
	logrus.Debug("Successfully generated new encryption key")
	return nil
}

func waitForLeader(keyFetcher security.KeyFetcher) error {
	var encryptionKey []byte
	for i := 0; i < 5; i++ {
		logrus.Debugf("Waiting for leader to generate encryption key...")
		encryptionKey, err := keyFetcher.GetEncryptionKey()
		if err != nil {
			return err
		}
		if len(encryptionKey) != 0 {
			return nil
		}
		time.Sleep(time.Microsecond * 100)
	}
	if len(encryptionKey) == 0 {
		return fmt.Errorf("Exceeded retry limit waiting for leader to generate encryption key")
	}
	return nil
}
