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
	"database/sql"
	"errors"
	"fmt"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
)

var ErrLockAcquisition = errors.New("failed to acquire lock")
var ErrUnlockAcquisition = errors.New("failed to unlock")

type Locker struct {
	*Storage
	isLocked      bool
	AdvisoryIndex int
	lockerCon     *sql.Conn
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (l *Locker) Lock(ctx context.Context) error {
	log.C(ctx).Debugf("Attempting to lock advisory lock with index (%d)", l.AdvisoryIndex)
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.isLocked || l.lockerCon != nil {
		log.C(ctx).Infof("Locker with advisory index (%d) is locked, so no attempt to lock it", l.AdvisoryIndex)
		return fmt.Errorf("lock is already acquired")
	}

	var err error
	if l.lockerCon, err = l.db.Conn(ctx); err != nil {
		return err
	}

	log.C(ctx).Debugf("Executing lock of locker with advisory index (%d)", l.AdvisoryIndex)
	rows, err := l.lockerCon.QueryContext(ctx, "SELECT pg_advisory_lock($1)", l.AdvisoryIndex)
	if err != nil {
		l.release(ctx)
		log.C(ctx).Infof("Failed to lock locker with advisory index (%d)", l.AdvisoryIndex)
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.C(ctx).WithError(err).Error("Could not close rows")
		}
	}()

	l.isLocked = true

	log.C(ctx).Debugf("Successfully locked locker with advisory index (%d)", l.AdvisoryIndex)
	return nil
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (l *Locker) TryLock(ctx context.Context) error {
	log.C(ctx).Debugf("Attempting to try_lock advisory lock with index (%d)", l.AdvisoryIndex)
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.isLocked || l.lockerCon != nil {
		log.C(ctx).Infof("Locker with advisory index (%d) is locked, so no attempt to try_lock it", l.AdvisoryIndex)
		return fmt.Errorf("try_lock is already acquired")
	}

	var err error
	if l.lockerCon, err = l.db.Conn(ctx); err != nil {
		return err
	}

	log.C(ctx).Debugf("Executing try_lock of locker with advisory index (%d)", l.AdvisoryIndex)
	rows, err := l.lockerCon.QueryContext(ctx, "SELECT pg_try_advisory_lock($1)", l.AdvisoryIndex)
	if err != nil {
		l.release(ctx)
		log.C(ctx).Debugf("Failed to try_lock locker with advisory index (%d)", l.AdvisoryIndex)
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.C(ctx).WithError(err).Error("Could not close rows")
		}
	}()

	var locked bool
	for rows.Next() {
		if err = rows.Scan(&locked); err != nil {
			l.release(ctx)
			return err
		}
	}

	if !locked {
		l.release(ctx)
		log.C(ctx).Debugf("Failed to try_lock locker with advisory index (%d) - either already locked or failed to lock", l.AdvisoryIndex)
		return ErrLockAcquisition
	}

	l.isLocked = true

	log.C(ctx).Debugf("Successfully try_locked locker with advisory index (%d)", l.AdvisoryIndex)
	return nil
}

// Unlock releases the database lock.
func (l *Locker) Unlock(ctx context.Context) error {
	log.C(ctx).Debugf("Attempting to unlock advisory lock with index (%d)", l.AdvisoryIndex)
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if !l.isLocked || l.lockerCon == nil {
		log.C(ctx).Infof("Locker with advisory index (%d) is not locked, so no attempt to unlock it", l.AdvisoryIndex)
		return nil
	}
	defer l.release(ctx)

	log.C(ctx).Debugf("Executing unlock of locker with advisory index (%d)", l.AdvisoryIndex)
	rows, err := l.lockerCon.QueryContext(ctx, "SELECT pg_advisory_unlock($1)", l.AdvisoryIndex)
	if err != nil {
		log.C(ctx).Infof("Failed to unlock locker with advisory index (%d)", l.AdvisoryIndex)
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.C(ctx).WithError(err).Error("Could not close rows")
		}
	}()

	var unlocked bool
	for rows.Next() {
		if err = rows.Scan(&unlocked); err != nil {
			return err
		}
	}

	if !unlocked {
		log.C(ctx).Infof("Failed to unlock locker with advisory index (%d) - either already unlocked or failed to unlock", l.AdvisoryIndex)
		return ErrUnlockAcquisition
	}

	l.isLocked = false

	log.C(ctx).Debugf("Successfully unlocked locker with advisory index (%d)", l.AdvisoryIndex)
	return nil
}

func (l *Locker) release(ctx context.Context) {
	if err := l.lockerCon.Close(); err != nil {
		log.C(ctx).WithError(err).Error("Could not release connection")
	}
	l.lockerCon = nil
}
