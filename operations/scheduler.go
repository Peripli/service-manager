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
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
)

type storageAction func(ctx context.Context, repository storage.Repository) (types.Object, error)

// Scheduler is responsible for storing Operation entities in the DB
// and also for spawning goroutines to execute the respective DB transaction asynchronously
type Scheduler struct {
	smCtx             context.Context
	repository        storage.TransactionalRepository
	workers           chan struct{}
	jobTimeout        time.Duration
	deletionTimeout   time.Duration
	reschedulingDelay time.Duration
	wg                *sync.WaitGroup
}

// NewScheduler constructs a Scheduler
func NewScheduler(smCtx context.Context, repository storage.TransactionalRepository, jobTimeout time.Duration, workerPoolSize int, wg *sync.WaitGroup) *Scheduler {
	return &Scheduler{
		smCtx:      smCtx,
		repository: repository,
		workers:    make(chan struct{}, workerPoolSize),
		jobTimeout: jobTimeout,
		wg:         wg,
	}
}

// ScheduleSyncStorageAction stores the job's Operation entity in DB and synchronously executes the CREATE/UPDATE/DELETE DB transaction
func (ds *Scheduler) ScheduleSyncStorageAction(ctx context.Context, operation *types.Operation, action storageAction) (types.Object, error) {
	log.C(ctx).Infof("Scheduling sync %s operation with id %s for resource of type %s with id %s", operation.Type, operation.ID, operation.ResourceType.String(), operation.ResourceID)

	if err := ds.executeOperationPreconditions(ctx, operation); err != nil {
		return nil, err
	}

	stateCtxWithOp, err := ds.addOperationToContext(ctx, operation)
	if err != nil {
		return nil, err
	}

	object, actionErr := action(stateCtxWithOp, ds.repository)
	if actionErr != nil {
		log.C(ctx).Errorf("failed to execute action for %s operation with id %s for %s entity with id %s: %s", operation.Type, operation.ID, operation.ResourceType, operation.ResourceID, actionErr)
	}

	if err := ds.handleActionResponse(&util.StateContext{Context: ctx}, actionErr, operation); err != nil {
		return nil, err
	}

	return object, nil
}

// ScheduleAsyncStorageAction stores the job's Operation entity in DB asynchronously executes the CREATE/UPDATE/DELETE DB transaction in a goroutine
func (ds *Scheduler) ScheduleAsyncStorageAction(ctx context.Context, operation *types.Operation, action storageAction) (err error) {
	select {
	case ds.workers <- struct{}{}:
		log.C(ctx).Infof("Scheduling async %s operation with id %s for resource of type %s with id %s", operation.Type, operation.ID, operation.ResourceType.String(), operation.ResourceID)
		if err := ds.executeOperationPreconditions(ctx, operation); err != nil {
			<-ds.workers
			return err
		}

		ds.wg.Add(1)
		go func(operation *types.Operation) {
			defer func() {
				if panicErr := recover(); panicErr != nil {
					err = fmt.Errorf("job panicked while executing: %s", panicErr)
					debug.PrintStack()
				}
				<-ds.workers
				ds.wg.Done()
			}()

			stateCtx := util.StateContext{Context: ctx}
			stateCtxWithOp, err := ds.addOperationToContext(stateCtx, operation)
			if err != nil {
				log.C(stateCtx).Error(err)
				return
			}

			stateCtxWithOpAndTimeout, cancel := context.WithTimeout(stateCtxWithOp, ds.jobTimeout)
			defer cancel()
			go func() {
				<-ds.smCtx.Done()
				cancel()
			}()

			var actionErr error
			if _, actionErr = action(stateCtxWithOpAndTimeout, ds.repository); actionErr != nil {
				log.C(stateCtx).Errorf("failed to execute action for %s operation with id %s for %s entity with id %s: %s", operation.Type, operation.ID, operation.ResourceType, operation.ResourceID, err)
			}

			if err := ds.handleActionResponse(stateCtx, actionErr, operation); err != nil {
				log.C(stateCtx).Error(err)
			}
		}(operation)
	default:
		log.C(ctx).Infof("Failed to schedule %s operation with id %s - all workers are busy.", operation.Type, operation.ID)
		return &util.HTTPError{
			ErrorType:   "ServiceUnavailable",
			Description: "Failed to schedule job. Server is busy - try again in a few minutes.",
			StatusCode:  http.StatusServiceUnavailable,
		}
	}

	return nil
}

func (ds *Scheduler) scheduleAsyncStorageActionDelayed(ctx context.Context, operation *types.Operation, action storageAction, delay time.Duration) (err error) {
	ds.wg.Add(1)
	go func() {
		defer ds.wg.Done()

		select {
		case <-ds.smCtx.Done():
		case <-time.After(delay):
			err = ds.ScheduleAsyncStorageAction(ctx, operation, action)
		}
	}()

	return err
}

func (ds *Scheduler) getLastOperation(ctx context.Context, operation *types.Operation) (*types.Operation, bool, error) {
	byResourceID := query.ByField(query.EqualsOperator, "resource_id", operation.ResourceID)
	orderDesc := query.OrderResultBy("paging_sequence", query.DescOrder)
	limitToOne := query.LimitResultBy(1)
	lastOperationObject, err := ds.repository.Get(ctx, types.OperationType, byResourceID, orderDesc, limitToOne)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return nil, false, nil
		}
		return nil, false, util.HandleStorageError(err, types.OperationType.String())
	}
	log.C(ctx).Infof("Last operation for resource with id %s of type %s is %+v", operation.ResourceID, operation.ResourceType, operation)

	return lastOperationObject.(*types.Operation), true, nil
}

func (ds *Scheduler) checkForConcurrentOperations(ctx context.Context, operation *types.Operation, lastOperation *types.Operation) error {
	log.C(ctx).Debugf("Checking if another operation is in progress to resource of type %s with id %s", operation.ResourceType.String(), operation.ResourceID)

	isDeletionInProgress := !lastOperation.DeletionScheduled.IsZero()
	isDeletionTimeoutExceeded := isDeletionInProgress && time.Now().After(lastOperation.DeletionScheduled.Add(ds.deletionTimeout))
	isOperationTimeoutNotExceeded := lastOperation.State == types.IN_PROGRESS && time.Now().Before(lastOperation.CreatedAt.Add(ds.jobTimeout))
	switch operation.Type {
	case types.CREATE:
		fallthrough
	case types.UPDATE:
		if isDeletionInProgress {
			<-ds.workers
			return &util.HTTPError{
				ErrorType:   "ConcurrentOperationInProgress",
				Description: "Deletion is currently in progress for this resource",
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}
		fallthrough
	case types.DELETE:
		if isDeletionTimeoutExceeded {
			<-ds.workers
			return &util.HTTPError{
				ErrorType:   "TimeoutExceeded",
				Description: "Deletion of this resource has been attempted for over the maximum timeout period. All operations will be rejected",
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}

		if isOperationTimeoutNotExceeded {
			<-ds.workers
			return &util.HTTPError{
				ErrorType:   "ConcurrentOperationInProgress",
				Description: "Another concurrent operation in progress for this resource",
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}
	}

	return nil
}

func (ds *Scheduler) storeOrUpdateOperation(ctx context.Context, operation, lastOperation *types.Operation) error {
	if lastOperation == nil || operation.ID != lastOperation.ID {
		log.C(ctx).Infof("Storing %s operation with id %s", operation.Type, operation.ID)
		if _, err := ds.repository.Create(ctx, operation); err != nil {
			return util.HandleStorageError(err, types.OperationType.String())
		}
	} else if operation.Type == lastOperation.Type && operation.Reschedule {
		log.C(ctx).Infof("Updating rescheduled %s operation with id %s", operation.Type, operation.ID)
		if _, err := ds.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
			return util.HandleStorageError(err, types.OperationType.String())
		}
	} else {
		return fmt.Errorf("operation with this id was already executed")
	}

	return nil
}

func fetchAndUpdateResource(ctx context.Context, repository storage.Repository, objectID string, objectType types.ObjectType, updateFunc func(obj types.Object)) error {
	byID := query.ByField(query.EqualsOperator, "id", objectID)
	objectFromDB, err := repository.Get(ctx, objectType, byID)
	if err != nil {
		return fmt.Errorf("failed to retrieve object of type %s with id %s:%s", objectType.String(), objectID, err)
	}

	updateFunc(objectFromDB)
	_, err = repository.Update(ctx, objectFromDB, query.LabelChanges{})
	if err != nil {
		return fmt.Errorf("failed to update object with type %s and id %s", objectType, objectID)
	}

	log.C(ctx).Infof("Successfully updated object of type %s and id %s ", objectType, objectID)
	return nil
}

func updateOperationState(ctx context.Context, repository storage.Repository, operation *types.Operation, state types.OperationState, opErr *OperationError) error {
	operation.State = state

	if opErr != nil {
		bytes, err := json.Marshal(opErr)
		if err != nil {
			return err
		}
		operation.Errors = json.RawMessage(bytes)
	}

	_, err := repository.Update(ctx, operation, query.LabelChanges{})
	if err != nil {
		return fmt.Errorf("failed to update state of operation with id %s to %s", operation.ID, state)
	}

	log.C(ctx).Infof("Successfully updated state of operation with id %s to %s", operation.ID, state)
	return nil
}

func (ds *Scheduler) refetchOperation(ctx context.Context, operation *types.Operation) (*types.Operation, error) {
	opObject, opErr := ds.repository.Get(ctx, types.OperationType, query.ByField(query.EqualsOperator, "id", operation.ID))
	if opErr != nil {
		opErr = fmt.Errorf("failed to re-fetch currently executing operation with id %s from db: %s", operation.ID, opErr)
		if err := updateOperationState(ctx, ds.repository, operation, types.FAILED, &OperationError{Message: "Internal Server Error"}); err != nil {
			return nil, fmt.Errorf("setting new operation state due to err %s failed: %s", opErr, err)
		}
		return nil, opErr
	}

	return opObject.(*types.Operation), nil
}

func (ds *Scheduler) handleActionResponse(ctx context.Context, jobError error, opBeforeJob *types.Operation) error {
	opAfterJob, err := ds.refetchOperation(ctx, opBeforeJob)
	if err != nil {
		return err
	}

	// if an action error has occurred we mark the operation as failed and check if deletion has to be scheduled
	if jobError != nil {
		if opErr := updateOperationState(ctx, ds.repository, opAfterJob, types.FAILED, &OperationError{Message: jobError.Error()}); opErr != nil {
			return fmt.Errorf("setting new operation state failed: %s", opErr)
		}

		// we want to schedule deletion if the operation is marked for deletion and the deletion timeout is not yet reached
		isDeleteRescheduleRequired := !opAfterJob.DeletionScheduled.IsZero() &&
			time.Now().Before(opAfterJob.DeletionScheduled.Add(ds.deletionTimeout)) &&
			opAfterJob.State != types.SUCCEEDED
		if isDeleteRescheduleRequired {
			deletionAction := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
				byID := query.ByField(query.EqualsOperator, "id", opAfterJob.ResourceID)
				err := repository.Delete(ctx, opAfterJob.ResourceType, byID)
				if err != nil {
					return nil, err
				}
				return nil, nil
			}

			log.C(ctx).Infof("Scheduling of required delete operation after actual operation with id %s failed", opAfterJob.ID)
			//recursive, no need to handle error
			if err := ds.scheduleAsyncStorageActionDelayed(ctx, opAfterJob, deletionAction, ds.reschedulingDelay); err != nil {
				return fmt.Errorf("scheduling of required deletion job failed with err: %s", err)
			}
		}
		return jobError
		// action did not return an error and operation does not need to be rescheduled so we mark it as success
	}
	if !opAfterJob.Reschedule {
		if err := ds.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
			log.C(ctx).Infof("Successfully executed %s operation with id %s for %s entity with id %s", opAfterJob.Type, opAfterJob.ID, opAfterJob.ResourceType, opAfterJob.ResourceID)
			if err := updateOperationState(ctx, storage, opAfterJob, types.SUCCEEDED, nil); err != nil {
				return err
			}
			if opAfterJob.Type != types.DELETE {
				if err := fetchAndUpdateResource(ctx, storage, opAfterJob.ResourceID, opAfterJob.ResourceType, func(obj types.Object) {
					obj.SetReady(true)
				}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("failed to update resource ready or operation state after a successfully executing operation with id %s: %s", opAfterJob.ID, err)
		}
		log.C(ctx).Infof("Successful executed operation with ID (%s)", opAfterJob.ID)
		// action did not return an error but required a reschedule so we keep it in progress
	} else {
		log.C(ctx).Infof("%s operation with id %s for %s entity with id %s is marked as requiring a reschedule and will be kept in progress", opAfterJob.Type, opAfterJob.ID, opAfterJob.ResourceType, opAfterJob.ResourceID)
	}

	return nil
}

func (ds *Scheduler) addOperationToContext(ctx context.Context, operation *types.Operation) (context.Context, error) {
	ctxWithOp, setCtxErr := SetInContext(ctx, operation)
	if setCtxErr != nil {
		setCtxErr = fmt.Errorf("failed to set operation in job context: %s", setCtxErr)
		if err := updateOperationState(ctx, ds.repository, operation, types.FAILED, &OperationError{Message: "Internal Server Error"}); err != nil {
			return nil, fmt.Errorf("setting new operation state due to err %s failed: %s", setCtxErr, err)
		}
		return nil, setCtxErr
	}

	return ctxWithOp, nil
}

func (ds *Scheduler) executeOperationPreconditions(ctx context.Context, operation *types.Operation) error {
	if operation.State == types.SUCCEEDED {
		return fmt.Errorf("scheduling for operations of type %s is not allowed", string(types.SUCCEEDED))
	}

	if err := operation.Validate(); err != nil {
		return fmt.Errorf("scheduled operation is not valid: %s", err)
	}

	lastOperation, found, err := ds.getLastOperation(ctx, operation)
	if err != nil {
		return err
	}

	if found {
		if err := ds.checkForConcurrentOperations(ctx, operation, lastOperation); err != nil {
			return err
		}
	}

	if err := ds.storeOrUpdateOperation(ctx, operation, lastOperation); err != nil {
		return err
	}

	return nil
}
