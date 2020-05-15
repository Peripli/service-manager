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
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
)

const (
	initialOperationsLockIndex = 200
	ZeroTime                   = "0001-01-01 00:00:00+00"
)

// maintainerFunctor represents a named maintainer function which runs over a pre-defined period
type maintainerFunctor struct {
	name     string
	interval time.Duration
	execute  func()
}

// Maintainer ensures that operations old enough are deleted
// and that no orphan operations are left in the DB due to crashes/restarts of SM
type Maintainer struct {
	smCtx      context.Context
	repository storage.Repository
	scheduler  *Scheduler

	settings *Settings
	wg       *sync.WaitGroup

	functors         []maintainerFunctor
	operationLockers map[string]storage.Locker
}

// NewMaintainer constructs a Maintainer
func NewMaintainer(smCtx context.Context, repository storage.TransactionalRepository, lockerCreatorFunc storage.LockerCreatorFunc, options *Settings, wg *sync.WaitGroup) *Maintainer {
	maintainer := &Maintainer{
		smCtx:      smCtx,
		repository: repository,
		scheduler:  NewScheduler(smCtx, repository, options, options.DefaultPoolSize, wg),
		settings:   options,
		wg:         wg,
	}

	maintainer.functors = []maintainerFunctor{
		{
			name:     "cleanupExternalOperations",
			execute:  maintainer.cleanupExternalOperations,
			interval: options.CleanupInterval,
		},
		{
			name:     "cleanupInternalSuccessfulOperations",
			execute:  maintainer.cleanupInternalSuccessfulOperations,
			interval: options.CleanupInterval,
		},
		{
			name:     "cleanupInternalFailedOperations",
			execute:  maintainer.cleanupInternalFailedOperations,
			interval: options.CleanupInterval,
		},
		{
			name:     "markStuckOperationsFailed",
			execute:  maintainer.markStuckOperationsFailed,
			interval: options.MaintainerRetryInterval,
		},
		{
			name:     "rescheduleUnfinishedOperations",
			execute:  maintainer.rescheduleUnfinishedOperations,
			interval: options.MaintainerRetryInterval,
		},
		{
			name:     "rescheduleOrphanMitigationOperations",
			execute:  maintainer.rescheduleOrphanMitigationOperations,
			interval: options.MaintainerRetryInterval,
		},
		{
			name: "pollCascadedOperations",
			execute: maintainer.pollCascadedOperations,
			interval: options.PollingInterval,
		},
	}

	operationLockers := make(map[string]storage.Locker)
	advisoryLockStartIndex := initialOperationsLockIndex
	for _, functor := range maintainer.functors {
		operationLockers[functor.name] = lockerCreatorFunc(advisoryLockStartIndex)
		advisoryLockStartIndex++
	}

	maintainer.operationLockers = operationLockers

	return maintainer
}

// Run starts the two recurring jobs responsible for cleaning up operations which are too old
// and deleting orphan operations
func (om *Maintainer) Run() {
	for _, functor := range om.functors {
		functor := functor
		maintainerFunc := func() {
			log.C(om.smCtx).Infof("Attempting to retrieve lock for maintainer functor (%s)", functor.name)
			err := om.operationLockers[functor.name].TryLock(om.smCtx)
			if err != nil {
				log.C(om.smCtx).Infof("Failed to retrieve lock for maintainer functor (%s): %s", functor.name, err)
				return
			}
			defer func() {
				if err := om.operationLockers[functor.name].Unlock(om.smCtx); err != nil {
					log.C(om.smCtx).Warnf("Could not unlock for maintainer functor (%s): %s", functor.name, err)
				}
			}()
			log.C(om.smCtx).Infof("Successfully retrieved lock for maintainer functor (%s)", functor.name)

			functor.execute()
		}

		go maintainerFunc()
		go om.processOperations(maintainerFunc, functor.name, functor.interval)
	}
}

func (om *Maintainer) processOperations(functor func(), functorName string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			func() {
				om.wg.Add(1)
				defer om.wg.Done()
				log.C(om.smCtx).Infof("Starting execution of maintainer functor (%s)", functorName)
				functor()
				log.C(om.smCtx).Infof("Finished execution of maintainer functor (%s)", functorName)
			}()
		case <-om.smCtx.Done():
			ticker.Stop()
			log.C(om.smCtx).Info("Server is shutting down. Stopping operations maintainer...")
			return
		}
	}
}

// cleanUpExternalOperations cleans up periodically all external operations which are older than some specified time
func (om *Maintainer) cleanupExternalOperations() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		query.ByField(query.NotEqualsOperator, "platform_id", types.SMPlatform),
		// check if operation hasn't been updated for the operation's maximum allowed time to live in DB
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.Lifespan))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil && err != util.ErrNotFoundInStorage {
		log.C(om.smCtx).Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.C(om.smCtx).Debug("Finished cleaning up external operations")
}

// cleanupInternalSuccessfulOperations cleans up all successful internal operations which are older than some specified time
func (om *Maintainer) cleanupInternalSuccessfulOperations() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.SUCCEEDED)),
		// check if operation hasn't been updated for the operation's maximum allowed time to live in DB
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.Lifespan))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil && err != util.ErrNotFoundInStorage {
		log.C(om.smCtx).Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.C(om.smCtx).Debug("Finished cleaning up successful internal operations")
}

// cleanupInternalFailedOperations cleans up all failed internal operations which are older than some specified time
func (om *Maintainer) cleanupInternalFailedOperations() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.FAILED)),
		query.ByField(query.EqualsOperator, "reschedule", "false"),
		query.ByField(query.EqualsOperator, "deletion_scheduled", ZeroTime),
		// check if operation hasn't been updated for the operation's maximum allowed time to live in DB
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.Lifespan))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil && err != util.ErrNotFoundInStorage {
		log.C(om.smCtx).Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.C(om.smCtx).Debug("Finished cleaning up failed internal operations")
}

// rescheduleUnfinishedOperations reschedules IN_PROGRESS operations which are reschedulable, not scheduled for deletion and no goroutine is processing at the moment
func (om *Maintainer) rescheduleUnfinishedOperations() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOperator, "reschedule", "true"),
		query.ByField(query.EqualsOperator, "deletion_scheduled", ZeroTime),
		// check if operation hasn't been updated for the operation's maximum allowed time to execute
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.ActionTimeout))),
		// check if operation is still eligible for processing
		query.ByField(query.GreaterThanOperator, "created_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.ReconciliationOperationTimeout))),
	}

	objectList, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.C(om.smCtx).Debugf("Failed to fetch unprocessed operations: %s", err)
		return
	}

	operations := objectList.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		logger := log.C(om.smCtx).WithField(log.FieldCorrelationID, operation.CorrelationID)
		ctx := log.ContextWithLogger(om.smCtx, logger)

		var action storageAction

		switch operation.Type {
		case types.CREATE:
			object, err := om.repository.Get(ctx, operation.ResourceType, query.ByField(query.EqualsOperator, "id", operation.ResourceID))
			if err != nil {
				logger.Warnf("Failed to fetch resource with ID (%s) for operation with ID (%s): %s", operation.ResourceID, operation.ID, err)
				break
			}

			action = func(ctx context.Context, repository storage.Repository) (types.Object, error) {
				object, err := repository.Create(ctx, object)
				return object, util.HandleStorageError(err, operation.ResourceType.String())
			}
		case types.UPDATE:
			byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)
			object, err := om.repository.Get(ctx, operation.ResourceType, byID)
			if err != nil {
				logger.Warnf("Failed to fetch resource with ID (%s) for operation with ID (%s): %s", operation.ResourceID, operation.ID, err)
				break
			}
			action = func(ctx context.Context, repository storage.Repository) (types.Object, error) {
				object, err := repository.Update(ctx, object, nil, byID)
				return object, util.HandleStorageError(err, operation.ResourceType.String())
			}
		case types.DELETE:
			byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)

			action = func(ctx context.Context, repository storage.Repository) (types.Object, error) {
				err := repository.Delete(ctx, operation.ResourceType, byID)
				if err != nil {
					if err == util.ErrNotFoundInStorage {
						return nil, nil
					}
					return nil, util.HandleStorageError(err, operation.ResourceType.String())
				}
				return nil, nil
			}
		}

		if err := om.scheduler.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
			logger.Warnf("Failed to reschedule unprocessed operation with ID (%s): %s", operation.ID, err)
		}

		logger.Debugf("Successfully rescheduled unfinished operation %+v", operation)
	}
}

func (om *Maintainer) pollCascadedOperations(){

}


// rescheduleOrphanMitigationOperations reschedules orphan mitigation operations which no goroutine is processing at the moment
func (om *Maintainer) rescheduleOrphanMitigationOperations() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.NotEqualsOperator, "deletion_scheduled", ZeroTime),
		query.ByField(query.NotEqualsOperator, "type", string(types.UPDATE)),
		// check if operation hasn't been updated for the operation's maximum allowed time to execute
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.ActionTimeout))),
		// check if operation is still eligible for processing
		query.ByField(query.GreaterThanOperator, "created_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.ReconciliationOperationTimeout))),
	}

	objectList, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.C(om.smCtx).Debugf("Failed to fetch unprocessed orphan mitigation operations: %s", err)
		return
	}

	operations := objectList.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		logger := log.C(om.smCtx).WithField(log.FieldCorrelationID, operation.CorrelationID)
		ctx := log.ContextWithLogger(om.smCtx, logger)

		byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)

		action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
			err := repository.Delete(ctx, operation.ResourceType, byID)
			if err != nil {
				if err == util.ErrNotFoundInStorage {
					return nil, nil
				}
				return nil, util.HandleStorageError(err, operation.ResourceType.String())
			}
			return nil, nil
		}

		if err := om.scheduler.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
			logger.Warnf("Failed to reschedule unprocessed orphan mitigation operation with ID (%s): %s", operation.ID, err)
		}

		logger.Debugf("Successfully rescheduled orphan mitigation operation %+v", operation)
	}
}

// markStuckOperationsFailed checks for operations which are stuck in state IN_PROGRESS, updates their status to FAILED and schedules a delete action
func (om *Maintainer) markStuckOperationsFailed() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOperator, "reschedule", "false"),
		query.ByField(query.EqualsOperator, "deletion_scheduled", ZeroTime),
		// check if operation hasn't been updated for the operation's maximum allowed time to execute
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.ActionTimeout))),
	}

	objectList, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.C(om.smCtx).Debugf("Failed to fetch stuck operations: %s", err)
		return
	}

	operations := objectList.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		logger := log.C(om.smCtx).WithField(log.FieldCorrelationID, operation.CorrelationID)

		operation.State = types.FAILED

		if operation.Type == types.CREATE || operation.Type == types.DELETE {
			operation.DeletionScheduled = time.Now()
		}

		if _, err := om.repository.Update(om.smCtx, operation, types.LabelChanges{}); err != nil {
			logger.Warnf("Failed to update orphan operation with ID (%s) state to FAILED: %s", operation.ID, err)
			continue
		}

		if operation.Type == types.CREATE || operation.Type == types.DELETE {
			byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)
			action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
				err := repository.Delete(ctx, operation.ResourceType, byID)
				if err != nil {
					if err == util.ErrNotFoundInStorage {
						return nil, nil
					}
					return nil, util.HandleStorageError(err, operation.ResourceType.String())
				}
				return nil, nil
			}

			if err := om.scheduler.ScheduleAsyncStorageAction(om.smCtx, operation, action); err != nil {
				logger.Warnf("Failed to schedule delete action for stuck operation with ID (%s): %s", operation.ID, err)
			}
		}
	}

	log.C(om.smCtx).Debug("Finished marking stuck operations as failed")
}
