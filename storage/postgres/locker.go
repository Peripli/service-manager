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
	"fmt"
)

type Locker struct {
	*Storage
	isLocked      bool
	AdvisoryIndex int
}

// Lock acquires a database lock so that only one process can manipulate the encryption key.
// Returns an error if the process has already acquired the lock
func (pl *Locker) Lock(ctx context.Context) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	if pl.isLocked {
		return fmt.Errorf("lock is already acquired")
	}
	if _, err := pl.db.ExecContext(ctx, "SELECT pg_try_advisory_lock($1)", pl.AdvisoryIndex); err != nil {
		return err
	}
	pl.isLocked = true

	return nil
}

// Unlock releases the database lock.
func (pl *Locker) Unlock(ctx context.Context) error {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	if !pl.isLocked {
		return nil
	}

	if _, err := pl.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", pl.AdvisoryIndex); err != nil {
		return err
	}
	pl.isLocked = false

	return nil
}
