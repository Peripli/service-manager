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
func NewScheduler(smCtx context.Context, repository storage.TransactionalRepository, settings *Settings, poolSize int, wg *sync.WaitGroup) *Scheduler {
	return &Scheduler{
		smCtx:             smCtx,
		repository:        repository,
		workers:           make(chan struct{}, poolSize),
		jobTimeout:        settings.JobTimeout,
		deletionTimeout:   settings.ScheduledDeletionTimeout,
		reschedulingDelay: settings.ReschedulingInterval,
		wg:                wg,
	}
}

// ScheduleSyncStorageAction stores the job's Operation entity in DB and synchronously executes the CREATE/UPDATE/DELETE DB transaction
func (ds *Scheduler) ScheduleSyncStorageAction(ctx context.Context, operation *types.Operation, action storageAction) (types.Object, error) {
	initialLogMessage(ctx, operation, false)

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

	if object, err = ds.handleActionResponse(&util.StateContext{Context: ctx}, object, actionErr, operation); err != nil {
		return nil, err
	}

	return object, nil
}

// ScheduleAsyncStorageAction stores the job's Operation entity in DB asynchronously executes the CREATE/UPDATE/DELETE DB transaction in a goroutine
func (ds *Scheduler) ScheduleAsyncStorageAction(ctx context.Context, operation *types.Operation, action storageAction) error {
	select {
	case ds.workers <- struct{}{}:
		initialLogMessage(ctx, operation, true)
		if err := ds.executeOperationPreconditions(ctx, operation); err != nil {
			<-ds.workers
			return err
		}

		ds.wg.Add(1)
		stateCtx := util.StateContext{Context: ctx}
		go func(operation *types.Operation) {
			defer func() {
				if panicErr := recover(); panicErr != nil {
					errMessage := fmt.Errorf("job panicked while executing: %s", panicErr)
					op, opErr := ds.refetchOperation(stateCtx, operation)
					if opErr != nil {
						errMessage = fmt.Errorf("%s: setting new operation state failed: %s ", errMessage, opErr)
					}

					if opErr := updateOperationState(stateCtx, ds.repository, op, types.FAILED, &util.HTTPError{
						ErrorType:   "InternalServerError",
						Description: "job interrupted",
						StatusCode:  http.StatusInternalServerError,
					}); opErr != nil {
						errMessage = fmt.Errorf("%s: setting new operation state failed: %s ", errMessage, opErr)
					}
					log.C(stateCtx).Errorf("panic error: %s", errMessage)
					debug.PrintStack()
				}
				<-ds.workers
				ds.wg.Done()
			}()

			stateCtxWithOp, err := ds.addOperationToContext(stateCtx, operation)
			if err != nil {
				log.C(stateCtx).Error(err)
				return
			}

			stateCtxWithOpAndTimeout, timeoutCtxCancel := context.WithTimeout(stateCtxWithOp, ds.jobTimeout)
			defer timeoutCtxCancel()
			go func() {
				select {
				case <-ds.smCtx.Done():
					timeoutCtxCancel()
				case <-stateCtxWithOpAndTimeout.Done():
				}

			}()

			var actionErr error
			var objectAfterAction types.Object
			if objectAfterAction, actionErr = action(stateCtxWithOpAndTimeout, ds.repository); actionErr != nil {
				log.C(stateCtx).Errorf("failed to execute action for %s operation with id %s for %s entity with id %s: %s", operation.Type, operation.ID, operation.ResourceType, operation.ResourceID, actionErr)
			}

			if _, err := ds.handleActionResponse(stateCtx, objectAfterAction, actionErr, operation); err != nil {
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

func (ds *Scheduler) getLastOperation(ctx context.Context, operation *types.Operation) (*types.Operation, bool, error) {
	byResourceID := query.ByField(query.EqualsOperator, "resource_id", operation.ResourceID)
	orderDesc := query.OrderResultBy("paging_sequence", query.DescOrder)
	lastOperationObject, err := ds.repository.Get(ctx, types.OperationType, byResourceID, orderDesc)
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

	isDeletionInProgress := lastOperation.Type == types.DELETE && lastOperation.State == types.IN_PROGRESS &&
		time.Now().Before(lastOperation.UpdatedAt.Add(ds.jobTimeout))

	isDeletionScheduled := !lastOperation.DeletionScheduled.IsZero()
	isDeletionTimeoutExceeded := (isDeletionInProgress && time.Now().After(lastOperation.CreatedAt.Add(ds.deletionTimeout))) ||
		(isDeletionScheduled && time.Now().After(lastOperation.DeletionScheduled.Add(ds.deletionTimeout)))

	// for the outside world job timeout would have expired if the last update happened > job timeout time ago (this is worst case)
	// an "old" updated_at means that for a while nobody was processing this operation
	isLastOperationTimeoutNotExceeded := lastOperation.State == types.IN_PROGRESS && time.Now().Before(lastOperation.UpdatedAt.Add(ds.jobTimeout))

	isAReschedule := lastOperation.Reschedule && operation.Reschedule

	// depending on the last executed operation on the resource and the currently executing operation we determine if the
	// currently executing operation should be allowed
	switch lastOperation.Type {
	case types.CREATE:
		switch operation.Type {
		case types.CREATE:
			// a create is in progress and operation timeout is not exceeded
			// the new op is a create with no deletion scheduled and is not reschedule, so fail

			// this means that when the last operation and the new operation which is either reschedulable or has a deletion scheduled
			// it is up to the client to make sure such operations do not overlap
			if isLastOperationTimeoutNotExceeded && !isDeletionScheduled && !isAReschedule {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Another concurrent operation in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}

			// deletion was scheduled but deletion timeout was exceeded so disallow further delete attempts
			if isDeletionTimeoutExceeded {
				return &util.HTTPError{
					ErrorType:   "TimeoutExceeded",
					Description: "Deletion of this resource has been attempted for over the maximum timeout period. All operations will be rejected",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.UPDATE:
			// a create is in progress and job timeout is not exceeded
			// the new op is an update - we don't allow updating something that is not yet created so fail
			if isLastOperationTimeoutNotExceeded {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Another concurrent operation in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.DELETE:
		// we allow deletes even if create is in progress
		default:
			// unknown operation type
			return fmt.Errorf("operation type %s is unknown type", operation.Type)
		}
	case types.UPDATE:
		switch operation.Type {
		case types.CREATE:
			// it doesnt really make sense to create something that was recently updated
			if isLastOperationTimeoutNotExceeded {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Another concurrent operation in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.UPDATE:
			// an update is in progress and job timeout is not exceeded
			// the new op is an update with no deletion scheduled and is not a reschedule, so fail

			// this means that when the last operation and the new operation which is either reschedulable or has a deletion scheduled
			// it is up to the client to make sure such operations do not overlap
			if isLastOperationTimeoutNotExceeded && !isDeletionScheduled && !isAReschedule {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Another concurrent operation in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}

			// deletion was scheduled but deletion timeout was exceeded so disallow further delete attempts
			if isDeletionTimeoutExceeded {
				return &util.HTTPError{
					ErrorType:   "TimeoutExceeded",
					Description: "Deletion of this resource has been attempted for over the maximum timeout period. All operations will be rejected",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.DELETE:
			// we allow deletes even if update is in progress
		default:
			// unknown operation type
			return fmt.Errorf("operation type %s is unknown type", operation.Type)
		}
	case types.DELETE:
		switch operation.Type {
		case types.CREATE:
			// if the last op is a delete in progress or if it has a deletion scheduled, creates are not allowed
			if isLastOperationTimeoutNotExceeded || isDeletionScheduled {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Deletion is currently in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.UPDATE:
			// if delete is in progress or delete is scheduled, updates are not allowed
			if isLastOperationTimeoutNotExceeded || isDeletionScheduled {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Deletion is currently in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.DELETE:
			// a delete is in progress and job timeout is not exceeded
			// the new op is a delete with no deletion scheduled and is not a reschedule, so fail

			// this means that when the last operation and the new operation which is either reschedulable or has a deletion scheduled
			// it is up to the client to make sure such operations do not overlap
			if isLastOperationTimeoutNotExceeded && !isDeletionScheduled && !isAReschedule {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Deletion is currently in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}

			// deletion was scheduled but deletion timeout was exceeded so disallow further delete attempts
			if isDeletionTimeoutExceeded {
				return &util.HTTPError{
					ErrorType:   "TimeoutExceeded",
					Description: "Deletion of this resource has been attempted for over the maximum timeout period. All operations will be rejected",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		default:
			// unknown operation type
			return fmt.Errorf("operation type %s is unknown type", operation.Type)
		}
	default:
		// unknown operation type
		return fmt.Errorf("operation type %s is unknown type", lastOperation.Type)
	}

	return nil
}

func (ds *Scheduler) storeOrUpdateOperation(ctx context.Context, operation, lastOperation *types.Operation) error {
	if lastOperation == nil || operation.ID != lastOperation.ID {
		log.C(ctx).Infof("Storing %s operation with id %s", operation.Type, operation.ID)
		if _, err := ds.repository.Create(ctx, operation); err != nil {
			return util.HandleStorageError(err, types.OperationType.String())
		}
	} else if operation.Type == lastOperation.Type && (operation.Reschedule || !operation.DeletionScheduled.IsZero()) {
		log.C(ctx).Infof("Updating rescheduled %s operation with id %s", operation.Type, operation.ID)
		if _, err := ds.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
			return util.HandleStorageError(err, types.OperationType.String())
		}
	} else {
		return fmt.Errorf("operation with this id was already executed")
	}

	return nil
}

func updateResource(ctx context.Context, repository storage.Repository, objectAfterAction types.Object, updateFunc func(obj types.Object)) (types.Object, error) {
	updateFunc(objectAfterAction)
	updatedObject, err := repository.Update(ctx, objectAfterAction, query.LabelChanges{})
	if err != nil {
		return nil, fmt.Errorf("failed to update object with type %s and id %s", objectAfterAction.GetType(), objectAfterAction.GetID())
	}

	log.C(ctx).Infof("Successfully updated object of type %s and id %s ", objectAfterAction.GetType(), objectAfterAction.GetID())
	return updatedObject, nil
}

func fetchAndUpdateResource(ctx context.Context, repository storage.Repository, objectID string, objectType types.ObjectType, updateFunc func(obj types.Object)) error {
	byID := query.ByField(query.EqualsOperator, "id", objectID)
	objectFromDB, err := repository.Get(ctx, objectType, byID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return nil
		}
		return fmt.Errorf("failed to retrieve object of type %s with id %s:%s", objectType.String(), objectID, err)
	}

	_, err = updateResource(ctx, repository, objectFromDB, updateFunc)
	return err
}

func updateOperationState(ctx context.Context, repository storage.Repository, operation *types.Operation, state types.OperationState, opErr error) error {
	operation.State = state
	operation.UpdatedAt = time.Now()

	if opErr != nil {
		httpError := util.ToHTTPError(ctx, opErr)
		bytes, err := json.Marshal(httpError)
		if err != nil {
			return err
		}

		if len(operation.Errors) == 0 {
			log.C(ctx).Debugf("setting error of operation with id %s to %s", operation.ID, httpError)
			operation.Errors = json.RawMessage(bytes)
		} else {
			log.C(ctx).Debugf("operation with id %s already has a root cause error %s. Current error %s will not be written", operation.ID, string(operation.Errors), httpError)
		}
	}

	// this also updates updated_at which serves as "reporting" that someone is working on the operation
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
		if err := updateOperationState(ctx, ds.repository, operation, types.FAILED, opErr); err != nil {
			return nil, fmt.Errorf("setting new operation state due to err %s failed: %s", opErr, err)
		}
		return nil, opErr
	}

	return opObject.(*types.Operation), nil
}

func (ds *Scheduler) handleActionResponse(ctx context.Context, actionObject types.Object, actionError error, opBeforeJob *types.Operation) (types.Object, error) {
	opAfterJob, err := ds.refetchOperation(ctx, opBeforeJob)
	if err != nil {
		return nil, err
	}

	// if an action error has occurred we mark the operation as failed and check if deletion has to be scheduled
	if actionError != nil {
		return nil, ds.handleActionResponseFailure(ctx, actionError, opAfterJob)
		// if no error occurred and op is not reschedulable (has finished), mark it as success
	} else if !opAfterJob.Reschedule {
		return ds.handleActionResponseSuccess(ctx, actionObject, opAfterJob)
	}

	log.C(ctx).Infof("%s operation with id %s for %s entity with id %s is marked as requiring a reschedule and will be kept in progress", opAfterJob.Type, opAfterJob.ID, opAfterJob.ResourceType, opAfterJob.ResourceID)
	// action did not return an error but required a reschedule so we keep it in progress
	return actionObject, nil
}

func (ds *Scheduler) handleActionResponseFailure(ctx context.Context, actionError error, opAfterJob *types.Operation) error {
	if err := ds.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		if opErr := updateOperationState(ctx, ds.repository, opAfterJob, types.FAILED, actionError); opErr != nil {
			return fmt.Errorf("setting new operation state failed: %s", opErr)
		}
		// after a failed FAILED CREATE operation, update the ready field to false
		if opAfterJob.Type == types.CREATE && opAfterJob.State == types.FAILED {
			if err := fetchAndUpdateResource(ctx, storage, opAfterJob.ResourceID, opAfterJob.ResourceType, func(obj types.Object) {
				obj.SetReady(false)
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// we want to schedule deletion if the operation is marked for deletion and the deletion timeout is not yet reached
	isDeleteRescheduleRequired := !opAfterJob.DeletionScheduled.IsZero() &&
		time.Now().UTC().Before(opAfterJob.DeletionScheduled.Add(ds.deletionTimeout)) &&
		opAfterJob.State != types.SUCCEEDED

	if isDeleteRescheduleRequired {
		deletionAction := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
			byID := query.ByField(query.EqualsOperator, "id", opAfterJob.ResourceID)
			err := repository.Delete(ctx, opAfterJob.ResourceType, byID)
			if err != nil {
				if err == util.ErrNotFoundInStorage {
					return nil, nil
				}
				return nil, util.HandleStorageError(err, opAfterJob.ResourceType.String())
			}
			return nil, nil
		}

		log.C(ctx).Infof("Scheduling of required delete operation after actual operation with id %s failed", opAfterJob.ID)
		// if deletion timestamp was set on the op, reschedule the same op with delete action
		reschedulingDelayTimeout := time.After(ds.reschedulingDelay)
		select {
		case <-ds.smCtx.Done():
			return fmt.Errorf("sm context canceled: %s", ds.smCtx.Err())
		case <-reschedulingDelayTimeout:
			if orphanMitigationErr := ds.ScheduleAsyncStorageAction(ctx, opAfterJob, deletionAction); orphanMitigationErr != nil {
				return &util.HTTPError{
					ErrorType:   "BrokerError",
					Description: fmt.Sprintf("job failed with %s and orphan mitigation failed with %s", actionError, orphanMitigationErr),
					StatusCode:  http.StatusBadGateway,
				}
			}
		}
	}
	return actionError
}

func (ds *Scheduler) handleActionResponseSuccess(ctx context.Context, actionObject types.Object, opAfterJob *types.Operation) (types.Object, error) {
	if err := ds.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		var finalState types.OperationState
		if opAfterJob.Type != types.DELETE && !opAfterJob.DeletionScheduled.IsZero() {
			// successful orphan mitigation for CREATE/UPDATE should still leave the operation as FAILED
			finalState = types.FAILED
		} else {
			// a delete that succeed or an orphan mitigation caused by a delete that succeeded are both successful deletions
			finalState = types.SUCCEEDED
			opAfterJob.Errors = json.RawMessage{}
		}

		// a non reschedulable operation has finished with no errors:
		// this can either be an actual operation or an orphan mitigation triggered by an actual operation error
		// in either case orphan mitigation needn't be scheduled any longer because being here means either an
		// actual operation finished with no errors or an orphan mitigation caused by an actual operation finished with no errors
		opAfterJob.DeletionScheduled = time.Time{}
		log.C(ctx).Infof("Successfully executed %s operation with id %s for %s entity with id %s", opAfterJob.Type, opAfterJob.ID, opAfterJob.ResourceType, opAfterJob.ResourceID)
		if err := updateOperationState(ctx, storage, opAfterJob, finalState, nil); err != nil {
			return err
		}

		// after a successful CREATE operation, update the ready field to true
		if opAfterJob.Type == types.CREATE && finalState == types.SUCCEEDED {
			var err error
			if actionObject, err = updateResource(ctx, storage, actionObject, func(obj types.Object) {
				obj.SetReady(true)
			}); err != nil {
				return err
			}
		}
		return nil

	}); err != nil {
		return nil, fmt.Errorf("failed to update resource ready or operation state after a successfully executing operation with id %s: %s", opAfterJob.ID, err)
	}
	log.C(ctx).Infof("Successful executed operation with ID (%s)", opAfterJob.ID)

	return actionObject, nil
}

func (ds *Scheduler) addOperationToContext(ctx context.Context, operation *types.Operation) (context.Context, error) {
	ctxWithOp, setCtxErr := SetInContext(ctx, operation)
	if setCtxErr != nil {
		setCtxErr = fmt.Errorf("failed to set operation in job context: %s", setCtxErr)
		if err := updateOperationState(ctx, ds.repository, operation, types.FAILED, setCtxErr); err != nil {
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
			log.C(ctx).Error("concurrent operation has been rejected: last operation is %+v, current operation is %+v and error is %s", lastOperation, operation, err)
			return err
		}
	}

	if err := ds.storeOrUpdateOperation(ctx, operation, lastOperation); err != nil {
		return err
	}

	return nil
}

func initialLogMessage(ctx context.Context, operation *types.Operation, async bool) {
	var logPrefix string
	if operation.Reschedule {
		logPrefix = "Reschduling (reschedule=true)"
	} else if !operation.DeletionScheduled.IsZero() {
		logPrefix = "Scheduling orphan mitigation"
	} else {
		logPrefix = "Scheduling new"
	}
	if async {
		logPrefix += " async"
	} else {
		logPrefix += " sync"
	}
	log.C(ctx).Infof("%s %s operation with id %s for resource of type %s with id %s", logPrefix, operation.Type, operation.ID, operation.ResourceType.String(), operation.ResourceID)

}
