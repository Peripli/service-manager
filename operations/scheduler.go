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

	"github.com/Peripli/service-manager/operations/opcontext"

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
	smCtx                          context.Context
	repository                     storage.TransactionalRepository
	workers                        chan struct{}
	actionTimeout                  time.Duration
	reconciliationOperationTimeout time.Duration
	cascadeOrphanMitigationTimeout time.Duration
	reschedulingDelay              time.Duration
	reschedulingLongDelay          time.Duration
	wg                             *sync.WaitGroup
}

// NewScheduler constructs a Scheduler
func NewScheduler(smCtx context.Context, repository storage.TransactionalRepository, settings *Settings, poolSize int, wg *sync.WaitGroup) *Scheduler {
	return &Scheduler{
		smCtx:                          smCtx,
		repository:                     repository,
		workers:                        make(chan struct{}, poolSize),
		actionTimeout:                  settings.ActionTimeout,
		reconciliationOperationTimeout: settings.ReconciliationOperationTimeout,
		cascadeOrphanMitigationTimeout: settings.CascadeOrphanMitigationTimeout,
		reschedulingDelay:              settings.ReschedulingInterval,
		reschedulingLongDelay:          settings.ReschedulingLongInterval,
		wg:                             wg,
	}
}

//Identifies the preferred execution mode and execute the storage action
func (s *Scheduler) ScheduleStorageAction(ctx context.Context, operation *types.Operation, action storageAction, isAsyncSupported bool) (types.Object, bool, error) {
	var object types.Object
	var err error

	if operation.Context.IsAsyncNotDefined && isAsyncSupported {
		object, err = s.ScheduleSyncStorageAction(ctx, operation, action)

		if err != nil {
			return nil, false, err
		}

		lastOperation, _, _, err := s.getResourceLastOperation(ctx, operation, false)
		if err != nil {
			return nil, false, err
		}

		if lastOperation.Reschedule {
			if err := s.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
				return nil, false, err
			}
			return nil, true, nil
		}

		return object, false, err
	}

	if operation.Context.Async {

		if !isAsyncSupported {
			return nil, false, &util.HTTPError{
				ErrorType:   "InvalidRequest",
				Description: fmt.Sprintf("requested api doesn't support asynchronous operations"),
				StatusCode:  http.StatusBadRequest,
			}
		}

		log.C(ctx).Debugf("Request will be executed asynchronously")
		if err := s.ScheduleAsyncStorageAction(ctx, operation, action); err != nil {
			return nil, true, err
		}
		return nil, true, nil
	}

	log.C(ctx).Debugf("Request will be executed synchronously")
	object, err = s.ScheduleSyncStorageAction(ctx, operation, action)
	return object, false, err
}

// ScheduleSyncStorageAction stores the job's Operation entity in DB and synchronously executes the CREATE/UPDATE/DELETE DB transaction
func (s *Scheduler) ScheduleSyncStorageAction(ctx context.Context, operation *types.Operation, action storageAction) (types.Object, error) {
	initialLogMessage(ctx, operation, false)

	if err := s.executeOperationPreconditions(ctx, operation); err != nil {
		return nil, err
	}

	ctxWithOp, err := s.addOperationToContext(ctx, operation)
	if err != nil {
		return nil, err
	}

	object, actionErr := action(ctxWithOp, s.repository)
	if actionErr != nil {
		log.C(ctx).Errorf("failed to execute action for %s operation with id %s for %s entity with id %s: %s", operation.Type, operation.ID, operation.ResourceType, operation.ResourceID, actionErr)
	}

	if object, err = s.handleActionResponse(&util.StateContext{Context: ctx}, object, actionErr, operation); err != nil {
		return nil, err
	}

	return object, nil
}

// ScheduleAsyncStorageAction stores the job's Operation entity in DB asynchronously executes the CREATE/UPDATE/DELETE DB transaction in a goroutine
func (s *Scheduler) ScheduleAsyncStorageAction(ctx context.Context, operation *types.Operation, action storageAction) error {
	select {
	case s.workers <- struct{}{}:
		initialLogMessage(ctx, operation, true)
		if err := s.executeOperationPreconditions(ctx, operation); err != nil {
			<-s.workers
			return err
		}

		s.wg.Add(1)
		stateCtx := util.StateContext{Context: ctx}
		go func(operation *types.Operation) {
			defer func() {
				if panicErr := recover(); panicErr != nil {
					errMessage := fmt.Errorf("job panicked while executing: %s", panicErr)
					op, opErr := s.refetchOperation(stateCtx, operation)
					if opErr != nil {
						errMessage = fmt.Errorf("%s: setting new operation state failed: %s ", errMessage, opErr)
					}

					if opErr := updateOperationState(stateCtx, s.repository, op, types.FAILED, &util.HTTPError{
						ErrorType:   "InternalServerError",
						Description: "job interrupted",
						StatusCode:  http.StatusInternalServerError,
					}); opErr != nil {
						errMessage = fmt.Errorf("%s: setting new operation state failed: %s ", errMessage, opErr)
					}
					log.C(stateCtx).Errorf("panic error: %s", errMessage)
					debug.PrintStack()
				}
				<-s.workers
				s.wg.Done()
			}()

			stateCtxWithOp, err := s.addOperationToContext(stateCtx, operation)
			if err != nil {
				log.C(stateCtx).Error(err)
				return
			}

			stateCtxWithOpAndTimeout, timeoutCtxCancel := context.WithTimeout(stateCtxWithOp, s.actionTimeout)
			defer timeoutCtxCancel()
			go func() {
				select {
				case <-s.smCtx.Done():
					timeoutCtxCancel()
				case <-stateCtxWithOpAndTimeout.Done():
				}

			}()

			var actionErr error
			var objectAfterAction types.Object
			if objectAfterAction, actionErr = action(stateCtxWithOpAndTimeout, s.repository); actionErr != nil {
				log.C(stateCtx).Errorf("failed to execute action for %s operation with id %s for %s entity with id %s: %s", operation.Type, operation.ID, operation.ResourceType, operation.ResourceID, actionErr)
			}

			if _, err := s.handleActionResponse(stateCtx, objectAfterAction, actionErr, operation); err != nil {
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

func (s *Scheduler) getResourceLastOperation(ctx context.Context, operation *types.Operation, checkForExistingOperation bool) (*types.Operation, bool, bool, error) {
	queryParams := map[string]interface{}{
		"id_list":       []string{operation.ResourceID},
		"resource_type": string(operation.ResourceType),
	}
	resourceLastOps, err := s.repository.QueryForList(
		ctx,
		types.OperationType,
		storage.QueryForLastOperationsPerResource,
		queryParams)

	if err != nil {
		return nil, false, false, util.HandleStorageError(err, types.OperationType.String())
	}
	if resourceLastOps.Len() == 0 {
		log.C(ctx).Debugf("Could not find last operation for resource with id %s and type %s in SMDB. Ignoring missing operation", operation.ResourceID, operation.ResourceType)
		return nil, false, false, nil
	}

	currentOperationExists := false
	if checkForExistingOperation {
		byID := query.ByField(query.EqualsOperator, "id", operation.GetID())
		count, err := s.repository.Count(ctx, types.OperationType, byID)
		if err != nil {
			return nil, false, false, util.HandleStorageError(err, types.OperationType.String())
		}
		currentOperationExists = count > 0
	}
	lastOperation := resourceLastOps.ItemAt(0).(*types.Operation)
	log.C(ctx).Infof("Last operation for resource with id %s of type %s is %+v", lastOperation.ResourceID, lastOperation.ResourceType, lastOperation)

	return lastOperation, true, currentOperationExists, nil
}

func (s *Scheduler) checkForConcurrentOperations(ctx context.Context, operation *types.Operation, lastOperation *types.Operation) error {
	log.C(ctx).Debugf("Checking if another operation is in progress to resource of type %s with id %s", operation.ResourceType.String(), operation.ResourceID)

	isDeletionScheduled := lastOperation.InOrphanMitigationState()

	// for the outside world job timeout would have expired if the last update happened > job timeout time ago (this is worst case)
	// an "old" updated_at means that for a while nobody was processing this operation
	isLastOpInProgress := lastOperation.State == types.IN_PROGRESS && time.Now().Before(lastOperation.UpdatedAt.Add(s.actionTimeout))

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
			if isLastOpInProgress && !isDeletionScheduled && !isAReschedule {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Another concurrent operation in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.UPDATE:
			// a create is in progress and job timeout is not exceeded
			// the new op is an update - we don't allow updating something that is not yet created so fail
			if isLastOpInProgress {
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
			if isLastOpInProgress {
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
			if isLastOpInProgress && !isDeletionScheduled && !isAReschedule {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Another concurrent operation in progress for this resource",
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
			if isLastOpInProgress || isDeletionScheduled {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Deletion is currently in progress for this resource",
					StatusCode:  http.StatusUnprocessableEntity,
				}
			}
		case types.UPDATE:
			// if delete is in progress or delete is scheduled, updates are not allowed
			if isLastOpInProgress || isDeletionScheduled {
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
			if isLastOpInProgress && !isDeletionScheduled && !isAReschedule {
				return &util.HTTPError{
					ErrorType:   "ConcurrentOperationInProgress",
					Description: "Deletion is currently in progress for this resource",
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

func (s *Scheduler) storeOrUpdateOperation(ctx context.Context, operation *types.Operation, currentOpExists bool) error {
	// if a new operation is scheduled we need to store it
	if !currentOpExists {
		log.C(ctx).Infof("Storing %s operation with id %s", operation.Type, operation.ID)
		if _, err := s.repository.Create(ctx, operation); err != nil {
			return util.HandleStorageError(err, types.OperationType.String())
		}
	} else {
		log.C(ctx).Infof("Updating rescheduled %s operation with id %s", operation.Type, operation.ID)
		if _, err := s.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
			return util.HandleStorageError(err, types.OperationType.String())
		}
	}

	return nil
}

func updateTransitiveResources(ctx context.Context, storage storage.Repository, resources []*types.RelatedType, updateFunc func(obj types.Object)) error {
	for _, trR := range resources {
		if trR.OperationType == types.CREATE {
			if err := fetchAndUpdateResource(ctx, storage, trR.ID, trR.Type, updateFunc); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateResource(ctx context.Context, repository storage.Repository, objectAfterAction types.Object, updateFunc func(obj types.Object)) (types.Object, error) {
	updateFunc(objectAfterAction)
	updatedObject, err := repository.Update(ctx, objectAfterAction, types.LabelChanges{})
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
			log.C(ctx).Debugf("Could not find resource with id %s and type %s in SMDB. Ignoring missing resource", objectID, objectType)
			return nil
		}
		return fmt.Errorf("failed to retrieve object of type %s with id %s:%s", objectType.String(), objectID, err)
	}

	_, err = updateResource(ctx, repository, objectFromDB, updateFunc)
	return err
}

func updateOperationState(ctx context.Context, repository storage.Repository, operation *types.Operation, state types.OperationState, opErr error) error {
	operation.State = state

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
	_, err := repository.Update(ctx, operation, types.LabelChanges{})
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			log.C(ctx).Debugf("Could not find and update operation for resource with id %s and type %s in SMDB. Ignoring missing operation", operation.ResourceID, operation.ResourceType)
			return nil
		}
		return fmt.Errorf("failed to update state of operation with id %s to %s: %s", operation.ID, state, err)
	}

	log.C(ctx).Infof("Successfully updated state of operation with id %s to %s", operation.ID, state)
	return nil
}

func (s *Scheduler) refetchOperation(ctx context.Context, operation *types.Operation) (*types.Operation, error) {
	opObject, opErr := s.repository.Get(ctx, types.OperationType, query.ByField(query.EqualsOperator, "id", operation.ID))
	if opErr != nil {
		opErr = fmt.Errorf("failed to re-fetch currently executing operation with id %s from db: %s", operation.ID, opErr)
		if err := updateOperationState(ctx, s.repository, operation, types.FAILED, opErr); err != nil {
			return nil, fmt.Errorf("setting new operation state due to err %s failed: %s", opErr, err)
		}
		return nil, opErr
	}

	return opObject.(*types.Operation), nil
}

func (s *Scheduler) handleActionResponse(ctx context.Context, actionObject types.Object, actionError error, opBeforeJob *types.Operation) (types.Object, error) {
	opAfterJob, err := s.refetchOperation(ctx, opBeforeJob)
	if err != nil {
		return nil, err
	}
	// Store the transitive resources in the refeched operation as they were added to the one in the context (opBeforeJob)
	opAfterJob.TransitiveResources = opBeforeJob.TransitiveResources
	// add the operation to context because we want to work with the refeched operation for further storage actions
	ctx, err = s.addOperationToContext(ctx, opAfterJob)
	if err != nil {
		return nil, err
	}

	// if an action error has occurred we mark the operation as failed and check if deletion has to be scheduled
	if actionError != nil {
		return nil, s.handleActionResponseFailure(ctx, actionError, opAfterJob)
		// if no error occurred and op is not reschedulable (has finished), mark it as success
	} else if !opAfterJob.Reschedule {
		return s.handleActionResponseSuccess(ctx, actionObject, opAfterJob)
	}

	log.C(ctx).Infof("%s operation with id %s for %s entity with id %s is marked as requiring a reschedule and will be kept in progress", opAfterJob.Type, opAfterJob.ID, opAfterJob.ResourceType, opAfterJob.ResourceID)
	// action did not return an error but required a reschedule so we keep it in progress
	return actionObject, nil
}

func (s *Scheduler) handleActionResponseFailure(ctx context.Context, actionError error, opAfterJob *types.Operation) error {
	if err := s.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		// after a failed FAILED CREATE operation, update the ready field to false
		if opAfterJob.Type == types.CREATE && opAfterJob.State == types.FAILED {
			if err := fetchAndUpdateResource(ctx, storage, opAfterJob.ResourceID, opAfterJob.ResourceType, func(obj types.Object) {
				obj.SetReady(false)
			}); err != nil {
				return err
			}

			if err := updateTransitiveResources(ctx, storage, opAfterJob.TransitiveResources, func(obj types.Object) {
				obj.SetReady(false)
			}); err != nil {
				return err
			}
		}

		newState := types.FAILED
		// if this is a force cascade action, we are trying to delete it directly from the database
		// in case we are failing to delete it the operation will be marked as failed
		if opAfterJob.IsForceDeleteCascadeOperation() && !opAfterJob.InOrphanMitigationState() {
			err := storage.Delete(ctx, opAfterJob.ResourceType, query.ByField(query.EqualsOperator, "id", opAfterJob.ResourceID))
			if err != nil && err != util.ErrNotFoundInStorage {
				log.C(ctx).Errorf("Failed to force delete %s resource with id %s: %s", opAfterJob.ResourceType, opAfterJob.ResourceID, err)
			} else {
				newState = types.SUCCEEDED
			}
		}

		if opErr := updateOperationState(ctx, storage, opAfterJob, newState, actionError); opErr != nil {
			return fmt.Errorf("setting new operation state failed: %s", opErr)
		}

		return nil
	}); err != nil {
		return err
	}

	// we want to schedule deletion if the operation is marked for deletion and the deletion timeout is not yet reached
	isDeleteRescheduleRequired := opAfterJob.InOrphanMitigationState() &&
		time.Now().UTC().Before(opAfterJob.DeletionScheduled.Add(s.reconciliationOperationTimeout)) &&
		opAfterJob.State != types.SUCCEEDED

	if isDeleteRescheduleRequired {
		deletionAction := func(ctx context.Context, repository storage.Repository) (types.Object, error) {
			byID := query.ByField(query.EqualsOperator, "id", opAfterJob.ResourceID)
			err := repository.Delete(ctx, opAfterJob.ResourceType, byID)
			if err != nil {
				if err == util.ErrNotFoundInStorage {
					log.C(ctx).Debugf("Could not find resource with id %s and type %s during delete action in SMDB. Ignoring missing resource", opAfterJob.ResourceID, opAfterJob.ResourceType)
					return nil, nil
				}
				return nil, util.HandleStorageError(err, opAfterJob.ResourceType.String())
			}
			return nil, nil
		}

		log.C(ctx).Infof("Scheduling of required delete operation after actual operation with id %s failed", opAfterJob.ID)
		// if deletion timestamp was set on the op, reschedule the same op with delete action and wait for reschedulingDelay time
		// so that we don't DOS the broker
		reschedulingDelayTimeout := time.After(s.reschedulingDelay)
		if time.Now().UTC().After(opAfterJob.DeletionScheduled.Add(s.actionTimeout * 2)) {
			reschedulingDelayTimeout = time.After(s.reschedulingLongDelay)
		}
		select {
		case <-s.smCtx.Done():
			return fmt.Errorf("sm context canceled: %s", s.smCtx.Err())
		case <-reschedulingDelayTimeout:
			if orphanMitigationErr := s.ScheduleAsyncStorageAction(ctx, opAfterJob, deletionAction); orphanMitigationErr != nil {
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

func (s *Scheduler) handleActionResponseSuccess(ctx context.Context, actionObject types.Object, opAfterJob *types.Operation) (types.Object, error) {
	if err := s.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		finalState := opAfterJob.State
		if opAfterJob.Type != types.DELETE && opAfterJob.InOrphanMitigationState() {
			// successful orphan mitigation for CREATE/UPDATE should still leave the operation as FAILED
			finalState = types.FAILED
		} else {
			// Guard to avoid set SUCCEEDED state on pending cascade operations
			if len(opAfterJob.CascadeRootID) == 0 || opAfterJob.State != types.PENDING {
				// a delete that succeed or an orphan mitigation caused by a delete that succeeded are both successful deletions
				finalState = types.SUCCEEDED
			}
			opAfterJob.Errors = json.RawMessage{}
		}

		// a non reschedulable operation has finished with no errors:
		// this can either be an actual operation or an orphan mitigation triggered by an actual operation error
		// in either case orphan mitigation needn't be scheduled any longer because being here means either an
		// actual operation finished with no errors or an orphan mitigation caused by an actual operation finished with no errors
		opAfterJob.DeletionScheduled = time.Time{}
		log.C(ctx).Infof("Successfully executed %s operation with id %s for %s entity with id %s", opAfterJob.Type, opAfterJob.ID, opAfterJob.ResourceType, opAfterJob.ResourceID)

		// after a successful CREATE operation, update the ready field to true
		if opAfterJob.Type == types.CREATE && finalState == types.SUCCEEDED {
			var err error
			if actionObject, err = updateResource(ctx, storage, actionObject, func(obj types.Object) {
				if serviceInstance, ok := obj.(*types.ServiceInstance); ok {
					serviceInstance.Usable = true
				}
				obj.SetReady(true)
			}); err != nil {
				return err
			}

			if err := updateTransitiveResources(ctx, storage, opAfterJob.TransitiveResources, func(obj types.Object) {
				obj.SetReady(true)
			}); err != nil {
				return err
			}
		}

		if err := updateOperationState(ctx, storage, opAfterJob, finalState, nil); err != nil {
			return err
		}

		return nil

	}); err != nil {
		return nil, fmt.Errorf("failed to update resource ready or operation state after a successfully executing operation with id %s: %s", opAfterJob.ID, err)
	}
	log.C(ctx).Infof("Successful executed operation with ID (%s)", opAfterJob.ID)

	return actionObject, nil
}

func (s *Scheduler) addOperationToContext(ctx context.Context, operation *types.Operation) (context.Context, error) {
	ctxWithOp, setCtxErr := opcontext.Set(ctx, operation)
	if setCtxErr != nil {
		setCtxErr = fmt.Errorf("failed to set operation in job context: %s", setCtxErr)
		if err := updateOperationState(ctx, s.repository, operation, types.FAILED, setCtxErr); err != nil {
			return nil, fmt.Errorf("setting new operation state due to err %s failed: %s", setCtxErr, err)
		}
		return nil, setCtxErr
	}

	return ctxWithOp, nil
}

func (s *Scheduler) validateOperationDoesNotExceedTimeouts(operation *types.Operation) error {
	if operation.CascadeRootID != "" && operation.InOrphanMitigationState() && time.Now().UTC().After(operation.CreatedAt.Add(s.cascadeOrphanMitigationTimeout)) {
		return &util.HTTPError{
			ErrorType:   "ManualActionRequired",
			Description: fmt.Sprintf("operations is older than %v and has exceed the maximmum cascade orphan mitigation timeout. Rootcause error: %s", s.cascadeOrphanMitigationTimeout, operation.Errors),
			StatusCode:  http.StatusUnprocessableEntity,
		}
	}
	if time.Now().UTC().After(operation.CreatedAt.Add(s.reconciliationOperationTimeout)) {
		return &util.HTTPError{
			ErrorType:   "ManualActionRequired",
			Description: fmt.Sprintf("operation is older than %v and has exceeded the maximum reconciliation timeout. Rootcause error: %s", s.reconciliationOperationTimeout, operation.Errors),
			StatusCode:  http.StatusUnprocessableEntity,
		}
	}
	return nil
}

func (s *Scheduler) executeOperationPreconditions(ctx context.Context, operation *types.Operation) error {
	if operation.State == types.SUCCEEDED ||
		(operation.State == types.FAILED && !operation.InOrphanMitigationState()) {
		return fmt.Errorf("scheduling for operations %+v is not allowed due to invalid state", operation)
	}

	// if operation has reached the maximum allowed timeout for auto rescheduling of operation actions
	// if cascade operation has reached the maximum allowed time for orphan mitigation
	err := s.validateOperationDoesNotExceedTimeouts(operation)
	if err != nil {
		operation.DeletionScheduled = time.Time{}
		if opErr := updateOperationState(ctx, s.repository, operation, types.FAILED, err); opErr != nil {
			return fmt.Errorf("failed to update error of operation with id %s to %s", operation.ID, err)
		}
		return err
	}

	if err := operation.Validate(); err != nil {
		return fmt.Errorf("scheduled operation is not valid: %s", err)
	}

	lastOperation, lastOpFound, currentOpExists, err := s.getResourceLastOperation(ctx, operation, true)
	if err != nil {
		return err
	}

	if lastOpFound {
		if err := s.checkForConcurrentOperations(ctx, operation, lastOperation); err != nil {
			log.C(ctx).Errorf("concurrent operation has been rejected: last operation is %+v, current operation is %+v and error is %s", lastOperation, operation, err)
			return err
		}

		// Block updates of service instances or bindings that were not created successfully
		if operation.Type == types.UPDATE {
			if lastOperation.Type == types.CREATE && lastOperation.State == types.FAILED {
				if operation.ResourceType == types.ServiceBindingType || operation.ResourceType == types.ServiceInstanceType {
					return &util.HTTPError{
						ErrorType:   "UpdateOperationIsNotAllowed",
						Description: "Update is not possible for this resource",
						StatusCode:  http.StatusForbidden,
					}
				}
			}
		}
	}

	if err := s.storeOrUpdateOperation(ctx, operation, currentOpExists); err != nil {
		return err
	}

	return nil
}

func initialLogMessage(ctx context.Context, operation *types.Operation, async bool) {
	var logPrefix string
	if operation.Reschedule {
		logPrefix = "Rescheduling (reschedule=true)"
	} else if operation.InOrphanMitigationState() {
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
