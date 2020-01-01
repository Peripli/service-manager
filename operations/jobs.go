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

// Job represents an ExecutableJob which is responsible for executing a C/U/D DB operation
type Job struct {
	operation     *types.Operation
	operationFunc func(ctx context.Context, repository storage.Repository) (types.Object, error)

	objectType types.ObjectType
	reqCtx     context.Context
}

// Execute executes a C/U/D DB operation
func (j *Job) Execute(ctxWithTimeout context.Context, repository storage.Repository) (string, error) {
	log.D().Debugf("Starting execution of %s operation with id (%s) for %s entity", j.operation.Type, j.operation.ID, j.objectType)
	var err error

	opCtx := util.StateContext{Context: j.reqCtx}
	reqCtx, reqCtxCancel := context.WithCancel(j.reqCtx)

	timedOut := false
	go func() {
		<-ctxWithTimeout.Done()
		reqCtxCancel()
		timedOut = true
	}()

	if _, err = j.operationFunc(reqCtx, repository); err != nil {
		log.D().Debugf("Failed to execute %s operation with id (%s) for %s entity", j.operation.Type, j.operation.ID, j.objectType)

		if timedOut {
			err = errors.New("job timed out")
		}

		if opErr := updateOperationState(opCtx, repository, j.operation.ID, types.FAILED, &OperationError{Message: err.Error()}); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", j.operation.ID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return j.operation.ID, err
	}

	log.D().Debugf("Successfully executed %s operation with id (%s) for %s entity", j.operation.Type, j.operation.ID, j.objectType)
	if err = updateOperationState(opCtx, repository, j.operation.ID, types.SUCCEEDED, nil); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", j.operation.ID, types.SUCCEEDED)
	}

	return j.operation.ID, err
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
