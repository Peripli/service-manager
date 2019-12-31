package operations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
)

// ExecutableJob represents a DB operation that has to be executed.
// After execution, the ID of the operation being executed and an error (if occurred) is returned.
type ExecutableJob interface {
	Execute(ctx context.Context, repository storage.Repository) (string, error)
}

// baseJob provides the base of what an ExecutableJob should contain
type baseJob struct {
	operationID string

	reqCtx       context.Context
	reqCtxCancel context.CancelFunc
}

// CreateJob represents an ExecutableJob which is responsible for executing a Create DB operation
type CreateJob struct {
	baseJob
	object types.Object
}

// UpdateJob represents an ExecutableJob which is responsible for executing an Update DB operation
type UpdateJob struct {
	baseJob
	object       types.Object
	labelChanges query.LabelChanges
	criteria     []query.Criterion
}

// DeleteJob represents an ExecutableJob which is responsible for executing a Delete DB operation
type DeleteJob struct {
	baseJob
	objectType types.ObjectType
	criteria   []query.Criterion
}

// Execute executes a Create DB operation
func (co *CreateJob) Execute(ctx context.Context, repository storage.Repository) (string, error) {
	log.D().Debugf("Starting execution of CREATE operation with id (%s) for entity %s", co.operationID, co.object.GetType())
	var err error
	opCtx := util.StateContext{Context: co.reqCtx}

	timedOut := false
	go func() {
		<-ctx.Done()
		co.reqCtxCancel()
		timedOut = true
	}()

	if _, err = repository.Create(co.reqCtx, co.object); err != nil {
		log.D().Debugf("Failed to execute CREATE operation with id (%s) for entity %s", co.operationID, co.object.GetType())

		if timedOut {
			err = errors.New("job timed out")
		}

		if opErr := updateOperationState(opCtx, repository, co.operationID, types.FAILED, &OperationError{Message: err.Error()}); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", co.operationID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return co.operationID, err
	}

	log.D().Debugf("Successfully executed CREATE operation with id (%s) for entity %s", co.operationID, co.object.GetType())
	if err = updateOperationState(opCtx, repository, co.operationID, types.SUCCEEDED, nil); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", co.operationID, types.SUCCEEDED)
	}

	return co.operationID, err
}

// Execute executes an Update DB operation
func (uo *UpdateJob) Execute(ctx context.Context, repository storage.Repository) (string, error) {
	log.D().Debugf("Starting execution of UPDATE operation with id (%s) for entity %s", uo.operationID, uo.object.GetType())
	var err error
	opCtx := util.StateContext{Context: uo.reqCtx}

	timedOut := false
	go func() {
		<-ctx.Done()
		uo.reqCtxCancel()
		timedOut = true
	}()

	if _, err = repository.Update(uo.reqCtx, uo.object, uo.labelChanges, uo.criteria...); err != nil {
		log.D().Debugf("Failed to execute UPDATE operation with id (%s) for entity %s", uo.operationID, uo.object.GetType())

		if timedOut {
			err = errors.New("job timed out")
		}

		if opErr := updateOperationState(opCtx, repository, uo.operationID, types.FAILED, &OperationError{Message: err.Error()}); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", uo.operationID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return uo.operationID, err
	}

	log.D().Debugf("Successfully executed UPDATE operation with id (%s) for entity %s", uo.operationID, uo.object.GetType())
	if err = updateOperationState(opCtx, repository, uo.operationID, types.SUCCEEDED, nil); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", uo.operationID, types.SUCCEEDED)
	}

	return uo.operationID, err
}

// Execute executes a Delete DB operation
func (do *DeleteJob) Execute(ctx context.Context, repository storage.Repository) (string, error) {
	log.D().Debugf("Starting execution of DELETE operation with id (%s) for entity %s", do.operationID, do.objectType)
	var err error
	opCtx := util.StateContext{Context: do.reqCtx}

	timedOut := false
	go func() {
		<-ctx.Done()
		do.reqCtxCancel()
		timedOut = true
	}()

	if err = repository.Delete(do.reqCtx, do.objectType, do.criteria...); err != nil {
		log.D().Debugf("Failed to execute DELETE operation with id (%s) for entity %s", do.operationID, do.objectType)

		if timedOut {
			err = errors.New("job timed out")
		}

		if opErr := updateOperationState(opCtx, repository, do.operationID, types.FAILED, &OperationError{Message: err.Error()}); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", do.operationID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return do.operationID, err
	}

	log.D().Debugf("Successfully executed DELETE operation with id (%s) for entity %s", do.operationID, do.objectType)
	if err = updateOperationState(opCtx, repository, do.operationID, types.SUCCEEDED, nil); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", do.operationID, types.SUCCEEDED)
	}

	return do.operationID, err
}

func updateOperationState(ctx context.Context, repository storage.Repository, operationID string, state types.OperationState, opErr *OperationError) error {
	operation, err := fetchOperation(ctx, repository, operationID)
	if err != nil {
		return err
	}

	operation.State = state

	if opErr != nil {
		bytes, err := json.Marshal(opErr)
		if err != nil {
			return err
		}
		operation.Errors = json.RawMessage(bytes)
	}

	_, err = repository.Update(ctx, operation, query.LabelChanges{})
	if err != nil {
		log.D().Debugf("Failed to update state of operation with id (%s) to %s", operationID, state)
		return err
	}

	log.D().Debugf("Successfully updated state of operation with id (%s) to %s", operationID, state)
	return nil
}

func fetchOperation(ctx context.Context, repository storage.Repository, operationID string) (*types.Operation, error) {
	byID := query.ByField(query.EqualsOperator, "id", operationID)
	objFromDB, err := repository.Get(ctx, types.OperationType, byID)
	if err != nil {
		log.D().Debugf("Failed to retrieve operation with id (%s)", operationID)
		return nil, err
	}

	return objFromDB.(*types.Operation), nil
}
