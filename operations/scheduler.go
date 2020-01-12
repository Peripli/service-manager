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

package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"net/http"
	"sync"
	"time"
)

// Scheduler is responsible for storing Operation entities in the DB
// and also for spawning goroutines to execute the respective DB transaction asynchronously
type Scheduler struct {
	smCtx      context.Context
	repository storage.Repository
	workers    chan struct{}
	jobTimeout time.Duration
	wg         *sync.WaitGroup
}

// NewScheduler constructs a Scheduler
func NewScheduler(smCtx context.Context, repository storage.Repository, jobTimeout time.Duration, workerPoolSize int, wg *sync.WaitGroup) *Scheduler {
	return &Scheduler{
		smCtx:      smCtx,
		repository: repository,
		workers:    make(chan struct{}, workerPoolSize),
		jobTimeout: jobTimeout,
		wg:         wg,
	}
}

// Schedule stores the Job's Operation entity in DB and spawns a goroutine to execute the CREATE/UPDATE/DELETE DB transaction asynchronously
func (ds *Scheduler) Schedule(job Job) (string, error) {
	log.D().Infof("Scheduling %s operation with id (%s)", job.Operation.Type, job.Operation.ID)
	select {
	case ds.workers <- struct{}{}:
		log.D().Infof("Storing %s operation with id (%s)", job.Operation.Type, job.Operation.ID)
		if _, err := ds.repository.Create(job.ReqCtx, job.Operation); err != nil {
			<-ds.workers
			return "", util.HandleStorageError(err, job.Operation.GetType().String())
		}

		ds.wg.Add(1)
		go func() {
			defer func() {
				<-ds.workers
				ds.wg.Done()
			}()

			ctxWithTimeout, cancel := context.WithTimeout(ds.smCtx, ds.jobTimeout)
			defer cancel()

			operationID, err := job.Execute(ctxWithTimeout, ds.repository)
			if err != nil {
				log.D().Debugf("Error occurred during execution of operation with ID (%s): %s", operationID, err.Error())
				return
			}
			log.D().Debugf("Successful executed operation with ID (%s)", operationID)
		}()
	default:
		log.D().Infof("Failed to schedule %s operation with id (%s) - all workers are busy.", job.Operation.Type, job.Operation.ID)
		return "", &util.HTTPError{
			ErrorType:   "ServiceUnavailable",
			Description: "Failed to schedule job. Server is busy - try again in a few minutes.",
			StatusCode:  http.StatusServiceUnavailable,
		}
	}

	return job.Operation.ID, nil
}
