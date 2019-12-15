package operations

import (
	"context"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type ExecutableJob interface {
	Execute(ctx context.Context, repository storage.Repository, results chan error)
}

type CreateJob struct {
	operationID string
	reqCtx      context.Context
	object      types.Object
}

type UpdateJob struct {
	operationID  string
	reqCtx       context.Context
	object       types.Object
	labelChanges query.LabelChanges
	criteria     []query.Criterion
}

type DeleteJob struct {
	operationID string
	reqCtx      context.Context
	objectType  types.ObjectType
	criteria    []query.Criterion
}

func (co *CreateJob) Execute(ctx context.Context, repository storage.Repository, errChan chan error) {
	var err error
	defer func() {
		errChan <- err
	}()

	if _, err = repository.Create(co.reqCtx, co.object); err != nil {
		log.D().Debugf("Failed to execute CREATE operation with id (%s) for entity %s", co.operationID, co.object.GetType())
		if opErr := updateOperationState(co.reqCtx, repository, co.operationID, types.FAILED); err != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", co.operationID, types.FAILED)
			err = errors.New(fmt.Sprintf("%s:%s", err, opErr))
		}
		return
	}

	log.D().Debugf("Successfully executed CREATE operation with id (%s) for entity %s", co.operationID, co.object.GetType())
	if err = updateOperationState(co.reqCtx, repository, co.operationID, types.SUCCEEDED); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", co.operationID, types.SUCCEEDED)
	}
}

func (uo *UpdateJob) Execute(ctx context.Context, repository storage.Repository, errChan chan error) {
	var err error
	defer func() {
		errChan <- err
	}()

	if _, err = repository.Update(uo.reqCtx, uo.object, uo.labelChanges, uo.criteria...); err != nil {
		log.D().Debugf("Failed to execute UPDATE operation with id (%s) for entity %s", uo.operationID, uo.object.GetType())
		if opErr := updateOperationState(uo.reqCtx, repository, uo.operationID, types.FAILED); err != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", uo.operationID, types.FAILED)
			err = errors.New(fmt.Sprintf("%s:%s", err, opErr))
		}
		return
	}

	log.D().Debugf("Successfully executed UPDATE operation with id (%s) for entity %s", uo.operationID, uo.object.GetType())
	if err = updateOperationState(uo.reqCtx, repository, uo.operationID, types.SUCCEEDED); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", uo.operationID, types.SUCCEEDED)
	}
}

func (do *DeleteJob) Execute(ctx context.Context, repository storage.Repository, errChan chan error) {
	var err error
	defer func() {
		errChan <- err
	}()

	if err = repository.Delete(do.reqCtx, do.objectType, do.criteria...); err != nil {
		log.D().Debugf("Failed to execute DELETE operation with id (%s) for entity %s", do.operationID, do.objectType)
		if opErr := updateOperationState(do.reqCtx, repository, do.operationID, types.FAILED); err != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", do.operationID, types.FAILED)
			err = errors.New(fmt.Sprintf("%s:%s", err, opErr))
		}
		return
	}

	log.D().Debugf("Successfully executed DELETE operation with id (%s) for entity %s", do.operationID, do.objectType)
	if err = updateOperationState(do.reqCtx, repository, do.operationID, types.SUCCEEDED); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", do.operationID, types.SUCCEEDED)
	}
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
