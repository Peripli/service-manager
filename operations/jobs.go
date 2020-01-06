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
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"time"
)

// Job represents an ExecutableJob which is responsible for executing a C/U/D DB operation
type Job struct {
	ReqCtx     context.Context
	ObjectType types.ObjectType

	Operation     *types.Operation
	OperationFunc func(ctx context.Context, repository storage.Repository) (types.Object, error)
}

// Execute executes a C/U/D DB operation
func (j *Job) Execute(ctxWithTimeout context.Context, repository storage.Repository) (string, error) {
	log.D().Debugf("Starting execution of %s operation with id (%s) for %s entity", j.Operation.Type, j.Operation.ID, j.ObjectType)
	var err error

	opCtx := util.StateContext{Context: j.ReqCtx}
	reqCtx, reqCtxCancel := context.WithCancel(j.ReqCtx)

	timedOut := false
	go func() {
		<-ctxWithTimeout.Done()
		reqCtxCancel()
		timedOut = true
	}()

	log.D().Debugf("Claiming %s operation with id (%s) for %s entity", j.Operation.Type, j.Operation.ID, j.ObjectType)
	if err = claimOperation(opCtx, repository, j.Operation.ID); err != nil {
		log.D().Debugf("Failed to claim operation with id (%s)", j.Operation.ID)
	}

	if _, err = j.OperationFunc(reqCtx, repository); err != nil {
		log.D().Debugf("Failed to execute %s operation with id (%s) for %s entity", j.Operation.Type, j.Operation.ID, j.ObjectType)

		if timedOut {
			err = errors.New("job timed out")
		}

		if opErr := updateOperationState(opCtx, repository, j.Operation.ID, types.FAILED, false, &OperationError{Message: err.Error()}); opErr != nil {
			log.D().Debugf("Failed to set state of operation with id (%s) to %s", j.Operation.ID, types.FAILED)
			err = fmt.Errorf("%s : %s", err, opErr)
		}
		return j.Operation.ID, err
	}

	log.D().Debugf("Successfully executed %s operation with id (%s) for %s entity", j.Operation.Type, j.Operation.ID, j.ObjectType)
	if err = updateOperationState(opCtx, repository, j.Operation.ID, types.SUCCEEDED, false, nil); err != nil {
		log.D().Debugf("Failed to set state of operation with id (%s) to %s", j.Operation.ID, types.SUCCEEDED)
	}

	return j.Operation.ID, err
}

func claimOperation(ctx context.Context, repository storage.Repository, operationID string) error {
	return updateOperationState(ctx, repository, operationID, types.IN_PROGRESS, true, nil)
}

func updateOperationState(ctx context.Context, repository storage.Repository, operationID string, state types.OperationState, claimOperation bool, opErr *OperationError) error {
	operation, err := fetchOperation(ctx, repository, operationID)
	if err != nil {
		return err
	}

	operation.State = state

	if claimOperation {
		operation.ClaimedAt = time.Now()
	}

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
