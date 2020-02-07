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

package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
)

const advisoryLockKey = "pg_try_advisory_lock"

var ErrLockAcquisition = errors.New("failed to acquire lock")

type Locker struct {
	*Storage
	isLocked      bool
	AdvisoryIndex int
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (l *Locker) Lock(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.isLocked {
		return fmt.Errorf("lock is already acquired")
	}
	if _, err := l.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", l.AdvisoryIndex); err != nil {
		return err
	}
	l.isLocked = true

	return nil
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (l *Locker) TryLock(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.isLocked {
		return fmt.Errorf("lock is already acquired")
	}

	rows, err := l.db.QueryxContext(ctx, "SELECT pg_try_advisory_lock($1)", l.AdvisoryIndex)
	if err != nil {
		return err
	}

	m := map[string]interface{}{}
	for rows.Next() {
		if err = rows.MapScan(m); err != nil {
			return err
		}
	}

	locked, found := m[advisoryLockKey]
	if !found || locked != true {
		return ErrLockAcquisition
	}

	l.isLocked = true

	return nil
}

// Unlock releases the database lock.
func (l *Locker) Unlock(ctx context.Context) error {
	log.C(ctx).Infof("Attempting to unlock advisory lock with index (%d)", l.AdvisoryIndex)
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if !l.isLocked {
		log.C(ctx).Infof("Locker with advisory index (%d) is not locked, so no attempt to unlock it", l.AdvisoryIndex)
		return nil
	}

	log.C(ctx).Infof("Executing unlock of locker with advisory index (%d)", l.AdvisoryIndex)
	if _, err := l.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", l.AdvisoryIndex); err != nil {
		log.C(ctx).Infof("Failed to unlock locker with advisory index (%d)", l.AdvisoryIndex)
		return err
	}
	l.isLocked = false

	log.C(ctx).Infof("Successfully unlocked locker with advisory index (%d)", l.AdvisoryIndex)
	return nil
}
