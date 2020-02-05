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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"golang.org/x/net/context"
	"time"
)

type ActionFunc func(ctx context.Context, repository storage.Repository) (types.Object, error)

// Maintainer ensures that operations old enough are deleted
// and that no orphan operations are left in the DB due to crashes/restarts of SM
type Maintainer struct {
	smCtx      context.Context
	repository storage.Repository
	scheduler  *Scheduler

	jobTimeout                     time.Duration
	operationExpirationTime        time.Duration
	reconciliationOperationTimeout time.Duration

	markOrphansInterval time.Duration
	cleanupInterval     time.Duration
}

// NewMaintainer constructs a Maintainer
func NewMaintainer(smCtx context.Context, repository storage.Repository, options *Settings) *Maintainer {
	// TODO: bootstrap scheduler & reconciliationOperationTimeout
	return &Maintainer{
		smCtx:                   smCtx,
		repository:              repository,
		jobTimeout:              options.JobTimeout,
		operationExpirationTime: options.ExpirationTime,
		markOrphansInterval:     options.MarkOrphansInterval,
		cleanupInterval:         options.CleanupInterval,
	}
}

// Run starts the two recurring jobs responsible for cleaning up operations which are too old
// and deleting orphan operations
func (om *Maintainer) Run() {
	// TODO: Should all maintainer funcs be run initially?
	om.cleanupExternalOperations()
	om.cleanupInternalSuccessfulOperations()
	om.cleanupInternalFailedOperations()

	om.rescheduleUnprocessedOperations()
	om.rescheduleOrphanMitigationOperations()
	om.markOrphanOperationsFailed()

	go om.processOperations(om.cleanupExternalOperations, om.cleanupInterval)
	go om.processOperations(om.cleanupInternalSuccessfulOperations, om.cleanupInterval)
	go om.processOperations(om.cleanupInternalFailedOperations, 2*om.cleanupInterval)

	go om.processOperations(om.rescheduleUnprocessedOperations, om.jobTimeout/2)
	go om.processOperations(om.rescheduleOrphanMitigationOperations, om.jobTimeout/2)
	go om.processOperations(om.markOrphanOperationsFailed, om.markOrphansInterval)
}

// TODO: Consider some kind of Maintainer Func abstraction
func (om *Maintainer) processOperations(maintainerFunc func(), interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			maintainerFunc()
		case <-om.smCtx.Done():
			ticker.Stop()
			log.C(om.smCtx).Info("Server is shutting down. Stopping operations maintainer...")
			return
		}
	}
}

// TODO: fix (specialize) maintainer func logs
// TODO: Leave out the last operation for each resource id

// cleanUpExternalOperations cleans up periodically all external operations which are older than some specified time
func (om *Maintainer) cleanupExternalOperations() {
	criteria := []query.Criterion{
		query.ByField(query.NotEqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.operationExpirationTime))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Finished cleaning up external operations")
}

// cleanupInternalSuccessfulOperations cleans up all successful internal operations which are older than some specified time
func (om *Maintainer) cleanupInternalSuccessfulOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.SUCCEEDED)),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.operationExpirationTime))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Finished cleaning up successful internal operations")
}

// cleanupInternalFailedOperations cleans up all failed internal operations which are older than some specified time
func (om *Maintainer) cleanupInternalFailedOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.FAILED)),
		query.ByField(query.EqualsOperator, "reschedule", "false"),
		query.ByField(query.EqualsOperator, "deletion_scheduled", "0001-01-01 00:00:00+00"),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.operationExpirationTime))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Finished cleaning up failed internal operations")
}

// rescheduleUnprocessedOperations reschedules IN_PROGRESS operations which are reschedulable, not scheduled for deletion and no goroutine is processing at the moment
func (om *Maintainer) rescheduleUnprocessedOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOperator, "reschedule", "true"),
		query.ByField(query.EqualsOperator, "deletion_scheduled", "0001-01-01 00:00:00+00"),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.jobTimeout))),
		query.ByField(query.GreaterThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.reconciliationOperationTimeout))),
	}
	// TODO: consider using timeAfter for readibility

	objectList, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.D().Debugf("Failed to fetch unprocessed operations: %s", err)
		return
	}

	operations := objectList.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)

		var action storageAction

		switch operation.Type {
		case types.CREATE:
			object, err := om.repository.Get(om.smCtx, operation.ResourceType, query.ByField(query.EqualsOperator, "id", operation.ResourceID))
			if err != nil {
				log.D().Errorf("Failed to fetch resource with ID (%s) for operation with ID (%s): %s", operation.ResourceID, operation.ID, err)
				return
			}

			action = func(ctx context.Context, repository storage.Repository) (types.Object, error) {
				object, err := repository.Create(ctx, object)
				return object, util.HandleStorageError(err, operation.ResourceType.String())
			}
			/*
				case types.UPDATE:
					action = func(ctx context.Context, repository storage.Repository) (types.Object, error) {
						object, err := repository.Update(ctx, objFromDB, labelChanges, criteria...)
						return object, util.HandleStorageError(err, operation.ResourceType.String())
					}
			*/
		case types.DELETE:
			byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)

			action = func(ctx context.Context, repository storage.Repository) (types.Object, error) {
				err := repository.Delete(ctx, operation.ResourceType, byID)
				return nil, util.HandleStorageError(err, operation.ResourceType.String())
			}
		}

		// TODO: which ctx should we use?
		if err := om.scheduler.ScheduleAsyncStorageAction(om.smCtx, operation, action); err != nil {
			log.D().Debugf("Failed to reschedule unprocessed operation with ID (%s): %s", operation.ID, err)
		}
	}

	log.D().Debug("Finished rescheduling unprocessed operations")
}

// rescheduleOrphanMitigationOperations reschedules orphan mitigation operations which no goroutine is processing at the moment
func (om *Maintainer) rescheduleOrphanMitigationOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.NotEqualsOperator, "deletion_scheduled", "0001-01-01 00:00:00+00"),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.jobTimeout))),
		query.ByField(query.GreaterThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.reconciliationOperationTimeout))),
	}

	objectList, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.D().Debugf("Failed to fetch unprocessed orphan mitigation operations: %s", err)
		return
	}

	operations := objectList.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)

		action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
			err := repository.Delete(ctx, operation.ResourceType, byID)
			return nil, util.HandleStorageError(err, operation.ResourceType.String())
		}

		// TODO: which ctx should we use?
		if err := om.scheduler.ScheduleAsyncStorageAction(om.smCtx, operation, action); err != nil {
			log.D().Debugf("Failed to reschedule unprocessed orphan mitigation operation with ID (%s): %s", operation.ID, err)
		}
	}

	log.D().Debug("Finished rescheduling unprocessed orphan mitigation operations")
}

// markOrphanOperationsFailed checks for operations which are stuck in state IN_PROGRESS, updates their status to FAILED and schedules a delete action
func (om *Maintainer) markOrphanOperationsFailed() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOperator, "reschedule", "false"),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.jobTimeout))),
	}

	objectList, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.D().Debugf("Failed to fetch orphan operations: %s", err)
		return
	}

	operations := objectList.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		operation.DeletionScheduled = time.Now()

		if _, err := om.repository.Update(om.smCtx, operation, query.LabelChanges{}); err != nil {
			log.D().Debugf("Failed to update orphan operation with ID (%s) state to FAILED: %s", operation.ID, err)
			continue
		}

		byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)
		action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
			err := repository.Delete(ctx, operation.ResourceType, byID)
			return nil, util.HandleStorageError(err, operation.ResourceType.String())
		}

		// TODO: which ctx should we use?
		if err := om.scheduler.ScheduleAsyncStorageAction(om.smCtx, operation, action); err != nil {
			log.D().Debugf("Failed to schedule delete action for operation with ID (%s): %s", operation.ID, err)
		}
	}

	log.D().Debug("Finished marking orphan operations as failed")
}
