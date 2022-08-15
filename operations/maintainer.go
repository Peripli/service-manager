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
	"fmt"
	"math"
	"sync"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
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
	smCtx                   context.Context
	repository              storage.TransactionalRepository
	scheduler               *Scheduler
	cascadePollingScheduler *Scheduler
	settings                *Settings
	wg                      *sync.WaitGroup
	functors                []maintainerFunctor
	operationLockers        map[string]storage.Locker
}

// NewMaintainer constructs a Maintainer
func NewMaintainer(smCtx context.Context, repository storage.TransactionalRepository, lockerCreatorFunc storage.LockerCreatorFunc, options *Settings, wg *sync.WaitGroup) *Maintainer {
	maintainer := &Maintainer{
		smCtx:                   smCtx,
		repository:              repository,
		scheduler:               NewScheduler(smCtx, repository, options, options.DefaultPoolSize, wg),
		cascadePollingScheduler: NewScheduler(smCtx, repository, options, options.DefaultCascadePollingPoolSize, wg),
		settings:                options,
		wg:                      wg,
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
			name:     "cleanupFinishedCascadeOperations",
			execute:  maintainer.CleanupFinishedCascadeOperations,
			interval: options.CleanupInterval,
		},
		{
			name:     "pollUpdateCascadeOperations",
			execute:  maintainer.PollUpdateCascadeOperations,
			interval: options.PollCascadeInterval,
		},
		{
			name:     "cleanupResourcelessOperations",
			execute:  maintainer.CleanupResourcelessOperations,
			interval: options.CleanupInterval,
		},
		{
			name:     "pollPendingCascadeOperations",
			execute:  maintainer.pollPendingCascadeOperations,
			interval: options.PollCascadeInterval,
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
			log.C(om.smCtx).Debugf("Attempting to retrieve lock for maintainer functor (%s)", functor.name)
			err := om.operationLockers[functor.name].TryLock(om.smCtx)
			if err != nil {
				log.C(om.smCtx).Debugf("Failed to retrieve lock for maintainer functor (%s): %s", functor.name, err)
				return
			}
			defer func() {
				if err := om.operationLockers[functor.name].Unlock(om.smCtx); err != nil {
					log.C(om.smCtx).Warnf("Could not unlock for maintainer functor (%s): %s", functor.name, err)
				}
			}()
			log.C(om.smCtx).Debugf("Successfully retrieved lock for maintainer functor (%s)", functor.name)

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
				log.C(om.smCtx).Debugf("Starting execution of maintainer functor (%s)", functorName)
				functor()
				log.C(om.smCtx).Debugf("Finished execution of maintainer functor (%s)", functorName)
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
		query.BySubquery(query.InSubqueryOperator, "id", storage.GetSubQuery(storage.QueryForAllNotLastOperationsPerResource)),
	}

	om.batchDeleteOperation(criteria, om.settings.DeleteOperationsBatchSize)

	log.C(om.smCtx).Debug("Finished cleaning up external operations")
}

func (om *Maintainer) PollUpdateCascadeOperations() {
	rootsCriteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "type", string(types.UPDATE)),
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOrNilOperator, "parent_id", ""),
		query.ByField(query.NotEqualsOperator, "cascade_root_id", ""),
	}

	roots, err := om.repository.List(om.smCtx, types.OperationType, rootsCriteria...)
	if err != nil {
		log.C(om.smCtx).Debugf("Failed to fetch finished cascade operations: %s", err)
		return
	}
	for i := 0; i < roots.Len(); i++ {
		root := roots.ItemAt(i).(*types.Operation)
		logger := log.C(om.smCtx).WithField(log.FieldCorrelationID, root.CorrelationID)
		subOperations, err := GetSubOperations(om.smCtx, root, om.repository)
		if err != nil {
			log.C(om.smCtx).Debugf("Failed to get sub operations: %s", err)
			return
		}

		byRootID := query.ByField(query.EqualsOperator, "cascade_root_id", root.GetID())
		if len(subOperations.FailedOperations) > 0 && len(subOperations.FailedOperations)+len(subOperations.SucceededOperations) == subOperations.AllOperationsCount {
			errorsJson, err := PrepareAggregatedErrorsArray(subOperations.FailedOperations, root.ResourceID, root.ResourceType)
			if err != nil {
				logger.Errorf("Couldn't aggregate errors for failed operation with id %s: %s", root.ResourceID, err)
			}

			root.Errors = errorsJson
			root.State = types.FAILED
			if _, err := om.repository.Update(om.smCtx, root, nil, byRootID); err != nil && err != util.ErrNotFoundInStorage {
				log.C(om.smCtx).Errorf("Failed to update root operation: %s", err)
			}
			log.C(om.smCtx).Debug("Finished updating a cascade update operation state")
		}

		if len(subOperations.SucceededOperations) == subOperations.AllOperationsCount {
			root.State = types.SUCCEEDED
			if _, err := om.repository.Update(om.smCtx, root, nil, byRootID); err != nil && err != util.ErrNotFoundInStorage {
				log.C(om.smCtx).Errorf("Failed to cleanup an instance update root operation: %s", err)
			}
			log.C(om.smCtx).Debug("Finished updating a cascade update operation state")
		}
	}
}

// cleanupFinishedCascadeOperations cleans up all successful/failed internal cascade operations which are older than some specified time
func (om *Maintainer) CleanupFinishedCascadeOperations() {
	currentTime := time.Now()
	rootsCriteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOrNilOperator, "parent_id", ""),
		query.ByField(query.NotEqualsOperator, "cascade_root_id", ""),
		query.ByField(query.InOperator, "state", string(types.SUCCEEDED), string(types.FAILED)),
		// check if operation hasn't been updated for the operation's maximum allowed time to live in DB//
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.Lifespan))),
		query.BySubquery(query.InSubqueryOperator, "id", storage.GetSubQuery(storage.QueryForAllNotLastOperationsPerResource)),
	}

	roots, err := om.repository.List(om.smCtx, types.OperationType, rootsCriteria...)
	if err != nil {
		log.C(om.smCtx).Debugf("Failed to fetch finished cascade operations: %s", err)
		return
	}
	for i := 0; i < roots.Len(); i++ {
		root := roots.ItemAt(i)
		byRootID := query.ByField(query.EqualsOperator, "cascade_root_id", root.GetID())
		if err := om.repository.Delete(om.smCtx, types.OperationType, byRootID); err != nil && err != util.ErrNotFoundInStorage {
			log.C(om.smCtx).Errorf("Failed to cleanup cascade operations: %s", err)
		}
	}
	log.C(om.smCtx).Debug("Finished cleaning up successful cascade operations")
}

// cleanupInternalCascadeOperations cleans up all finished internal cascade operations which are older than some specified time
func (om *Maintainer) cleanupInternalSuccessfulOperations() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.SUCCEEDED)),
		// ignore cascade operations
		query.ByField(query.EqualsOrNilOperator, "cascade_root_id", ""),
		// check if operation hasn't been updated for the operation's maximum allowed time to live in DB
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.Lifespan))),
		query.BySubquery(query.InSubqueryOperator, "id", storage.GetSubQuery(storage.QueryForAllNotLastOperationsPerResource)),
	}

	om.batchDeleteOperation(criteria, om.settings.DeleteOperationsBatchSize)

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
		// ignore cascade operations
		query.ByField(query.EqualsOrNilOperator, "cascade_root_id", ""),
		// check if operation hasn't been updated for the operation's maximum allowed time to live in DB
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.Lifespan))),
		query.BySubquery(query.InSubqueryOperator, "id", storage.GetSubQuery(storage.QueryForAllNotLastOperationsPerResource)),
	}

	om.batchDeleteOperation(criteria, om.settings.DeleteOperationsBatchSize)

	log.C(om.smCtx).Debug("Finished cleaning up failed internal operations")
}

func (om *Maintainer) CleanupResourcelessOperations() {
	currentTime := time.Now()
	criteria := []query.Criterion{
		// check if operation hasn't been updated for the operation's maximum allowed time to live in DB
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(currentTime.Add(-om.settings.Lifespan))),
	}
	// Build the 'WHERE' clause of the deletion statement with 'byIdNotExist' criterion for each resource table.
	for _, entity := range om.repository.GetEntities() {
		if entity.TableName == "operations" {
			continue
		}
		params := storage.SubQueryParams{
			"RESOURCE_TABLE": entity.TableName,
		}
		subQuery, err := storage.GetSubQueryWithParams(storage.QueryForOperationsWithResource, params)
		if err != nil {
			log.C(om.smCtx).Debugf(
				"Failed resolving template parameters for sub-query: %v. Error: %v",
				storage.QueryForOperationsWithResource,
				err)
		}
		byIDNotExistCriterion := query.ByNotExists(subQuery)
		criteria = append(criteria, byIDNotExistCriterion)
	}
	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil && err != util.ErrNotFoundInStorage {
		log.C(om.smCtx).Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.C(om.smCtx).Debug("Finished cleaning up resource-less operations")
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

func (om *Maintainer) pollPendingCascadeOperations() {
	criteria := []query.Criterion{
		query.ByField(query.NotEqualsOperator, "cascade_root_id", ""),
		query.ByField(query.EqualsOperator, "type", string(types.DELETE)),
		query.ByField(query.EqualsOperator, "state", string(types.PENDING)),
	}
	operations, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.C(om.smCtx).Errorf("Failed to fetch cascaded operations in progress: %s", err)
		return
	}

	skipSameResourcesForCurrentIteration := make(map[string]bool)
	operations = operations.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		if skipSameResourcesForCurrentIteration[operation.ResourceID] {
			continue
		}
		logger := log.C(om.smCtx).WithField(log.FieldCorrelationID, operation.CorrelationID)
		ctx := log.ContextWithLogger(om.smCtx, logger)

		subOperations, err := GetSubOperations(ctx, operation, om.repository)
		if err != nil {
			logger.Errorf("Failed to retrieve children of the operation with ID (%s): %s", operation.ID, err)
			continue
		}

		if subOperations.AllOperationsCount == len(subOperations.SucceededOperations) {
			if types.IsVirtualType(operation.ResourceType) {
				operation.State = types.SUCCEEDED
				if _, err := om.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
					logger.Errorf("Failed to update the operation with ID (%s) state to Success: %s", operation.ID, err)
				}
			} else {
				sameResourceState, skip, err := handleDuplicateOperations(ctx, om.repository, operation)
				if err != nil {
					logger.Errorf("Failed to validate if operation with ID (%s) is in polling: %s", operation.ID, err)
					continue
				}
				if skip {
					skipSameResourcesForCurrentIteration[operation.ResourceID] = true
					continue
				}
				if sameResourceState != "" {
					operation.State = sameResourceState
					if _, err := om.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
						logger.Errorf("Failed to update the operation with ID (%s) state to Success: %s", operation.ID, err)
						continue
					}
				} else {
					operation.State = types.IN_PROGRESS
					action := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
						byID := query.ByField(query.EqualsOperator, "id", operation.ResourceID)
						err := repository.Delete(ctx, operation.ResourceType, byID)
						if err != nil {
							if err == util.ErrNotFoundInStorage {
								return nil, nil
							}
							return nil, util.HandleStorageError(err, operation.ResourceType.String())
						}
						return nil, nil
					}
					if err := om.cascadePollingScheduler.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
						logger.Errorf("Failed to reschedule cascaded delete operation with ID (%s): %s ok for concurrent deletion failure", operation.ID, err)
					} else {
						skipSameResourcesForCurrentIteration[operation.ResourceID] = true
					}
				}
			}
		} else if len(subOperations.FailedOperations) > 0 && len(subOperations.FailedOperations)+len(subOperations.SucceededOperations) == subOperations.AllOperationsCount {
			if !operation.IsForceDeleteCascadeOperation() {
				// if its not force cascade delete operion -> aggregating errors
				errorsJson, err := PrepareAggregatedErrorsArray(subOperations.FailedOperations, operation.ResourceID, operation.ResourceType)
				if err != nil {
					logger.Errorf("Couldn't aggregate errors for failed operation with id %s: %s", operation.ResourceID, err)
				} else {
					operation.Errors = errorsJson
				}
			} else if operation.CascadeRootID == operation.ID {
				// if operation is force cascade delete and its failed -> database error
				operation.Errors = []byte(fmt.Sprintf(`{"error": "InternalServerError", "description": "failed to force cascade delete of %s with id %s"}`, operation.ResourceType, operation.ResourceID))
			}

			operation.State = types.FAILED
			if _, err := om.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
				logger.Errorf("Failed to update the operation with ID (%s) state to Success: %s", operation.ID, err)
			}
		}
	}
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

func (om *Maintainer) batchDeleteOperation(criteria []query.Criterion, batchSize int) {
	numberOfOperationsToDelete, err := om.repository.Count(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.C(om.smCtx).Errorf("Error on cleanup operations - get number of operation to delete failed: %s", err)
		return
	}
	criteria = append(criteria, query.LimitResultBy(batchSize))
	for i := 0.0; i < math.Ceil(float64(numberOfOperationsToDelete)/float64(batchSize)); i++ {
		operations, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
		if err != nil {
			log.C(om.smCtx).Errorf("Failed to cleanup operations - list query failed: %s", err)
			return
		}
		if operations.Len() == 0 {
			log.C(om.smCtx).Errorf("Failed to cleanup operations - list query returned 0 operations but numberOfOperationsToDelete is %d", numberOfOperationsToDelete)
			return
		}
		operationIDs := make([]string, 0, operations.Len())
		for i := 0; i < operations.Len(); i++ {
			operationIDs = append(operationIDs, operations.ItemAt(i).GetID())
		}
		if err := om.repository.Delete(om.smCtx, types.OperationType, query.ByField(query.InOperator, "id", operationIDs...)); err != nil && err != util.ErrNotFoundInStorage {
			log.C(om.smCtx).Errorf("Failed to cleanup operations - delete query failed: %s", err)
			return
		}
	}
}
