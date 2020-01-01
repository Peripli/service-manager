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
	"strings"
	"time"
)

// DefaultScheduler implements JobScheduler interface. It's responsible for
// storing C/U/D jobs so that a worker pool can eventually start consuming these jobs
type DefaultScheduler struct {
	smCtx      context.Context
	repository storage.Repository
	jobTimeout time.Duration
}

// NewScheduler constructs a DefaultScheduler
func NewScheduler(smCtx context.Context, repository storage.Repository, jobTimeout time.Duration) *DefaultScheduler {
	return &DefaultScheduler{
		smCtx:      smCtx,
		repository: repository,
		jobTimeout: jobTimeout,
	}
}

type Job struct {
	ReqCtx     context.Context
	ObjectType types.ObjectType

	Operation     *types.Operation
	OperationFunc func(ctx context.Context, repository storage.Repository) (types.Object, error)
}

// Schedule stores a CREATE/UPDATE/DELETE operation and then processes it asynchronously
func (ds *DefaultScheduler) Schedule(job *Job) (string, error) {
	log.D().Infof("Storing %s operation with id (%s)", job.Operation.Type, job.Operation.ID)
	if _, err := ds.repository.Create(job.ReqCtx, job.Operation); err != nil {
		return "", util.HandleStorageError(err, job.Operation.GetType().String())
	}

	log.D().Infof("Scheduling %s operation with id (%s)", job.Operation.Type, job.Operation.ID)
	go func() {
		log.D().Debugf("Starting execution of %s operation with id (%s) for %s entity", job.Operation.Type, job.Operation.ID, job.ObjectType)
		var err error

		opCtx := util.StateContext{Context: job.ReqCtx}
		reqCtx, reqCtxCancel := context.WithTimeout(job.ReqCtx, 1*time.Minute)
		defer reqCtxCancel()

		if _, err = job.OperationFunc(reqCtx, ds.repository); err != nil {
			log.D().Debugf("Failed to execute %s operation with id (%s) for %s entity", job.Operation.Type, job.Operation.ID, job.ObjectType)

			if strings.Contains(err.Error(), "timed out") {
				err = errors.New("job timed out")
			}

			if opErr := updateOperationState(opCtx, ds.repository, job.Operation.ID, types.FAILED, &OperationError{Message: err.Error()}); opErr != nil {
				log.D().Debugf("Failed to set state of operation with id (%s) to %s", job.Operation.ID, types.FAILED)
				err = fmt.Errorf("%s : %s", err, opErr)
			}
		}

		log.D().Debugf("Successfully executed %s operation with id (%s) for %s entity", job.Operation.Type, job.Operation.ID, job.ObjectType)
		if err = updateOperationState(opCtx, ds.repository, job.Operation.ID, types.SUCCEEDED, nil); err != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", job.Operation.ID, types.SUCCEEDED)
		}
	}()

	return job.Operation.ID, nil
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
