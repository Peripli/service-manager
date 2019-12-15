package operations

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
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
	var err error

	go func() {
		<-ctx.Done()
		co.reqCtxCancel()
	}()

	if _, err = repository.Create(co.reqCtx, co.object); err != nil {
		log.D().Debugf("Failed to execute CREATE operation with id (%s) for entity %s", co.operationID, co.object.GetType())

		if opErr := updateOperationState(co.reqCtx, repository, co.operationID, types.FAILED); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", co.operationID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return co.operationID, err
	}

	log.D().Debugf("Successfully executed CREATE operation with id (%s) for entity %s", co.operationID, co.object.GetType())
	if err = updateOperationState(co.reqCtx, repository, co.operationID, types.SUCCEEDED); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", co.operationID, types.SUCCEEDED)
	}

	return co.operationID, err
}

// Execute executes an Update DB operation
func (uo *UpdateJob) Execute(ctx context.Context, repository storage.Repository) (string, error) {
	var err error

	go func() {
		<-ctx.Done()
		uo.reqCtxCancel()
	}()

	if _, err = repository.Update(uo.reqCtx, uo.object, uo.labelChanges, uo.criteria...); err != nil {
		log.D().Debugf("Failed to execute UPDATE operation with id (%s) for entity %s", uo.operationID, uo.object.GetType())

		if opErr := updateOperationState(uo.reqCtx, repository, uo.operationID, types.FAILED); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", uo.operationID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return uo.operationID, err
	}

	log.D().Debugf("Successfully executed UPDATE operation with id (%s) for entity %s", uo.operationID, uo.object.GetType())
	if err = updateOperationState(uo.reqCtx, repository, uo.operationID, types.SUCCEEDED); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", uo.operationID, types.SUCCEEDED)
	}

	return uo.operationID, err
}

// Execute executes a Delete DB operation
func (do *DeleteJob) Execute(ctx context.Context, repository storage.Repository) (string, error) {
	var err error

	go func() {
		<-ctx.Done()
		do.reqCtxCancel()
	}()

	if err = repository.Delete(do.reqCtx, do.objectType, do.criteria...); err != nil {
		log.D().Debugf("Failed to execute DELETE operation with id (%s) for entity %s", do.operationID, do.objectType)

		if opErr := updateOperationState(do.reqCtx, repository, do.operationID, types.FAILED); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", do.operationID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return do.operationID, err
	}

	log.D().Debugf("Successfully executed DELETE operation with id (%s) for entity %s", do.operationID, do.objectType)
	if err = updateOperationState(do.reqCtx, repository, do.operationID, types.SUCCEEDED); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", do.operationID, types.SUCCEEDED)
	}

	return do.operationID, err
}

func updateOperationState(ctx context.Context, repository storage.Repository, operationID string, state types.OperationState) error {
	operation, err := fetchOperation(ctx, repository, operationID)
	if err != nil {
		return err
	}

	operation.State = state

	_, err = repository.Update(ctx, operation, query.LabelChanges{})
	if err != nil {
		log.D().Debugf("Failed to update state of operation with id (%s) to SUCCEEDED", operationID)
		return err
	}

	log.D().Debugf("Successfully updated state of operation with id (%s) to SUCCEEDED", operationID)
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
