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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"golang.org/x/net/context"
	"time"
)

// Maintainer ensures that operations old enough are deleted
// and that no orphan operations are left in the DB due to crashes/restarts of SM
type Maintainer struct {
	smCtx      context.Context
	repository storage.Repository

	jobTimeout              time.Duration
	operationExpirationTime time.Duration

	markOrphansInterval time.Duration
	cleanupInterval     time.Duration
}

// NewMaintainer constructs a Maintainer
func NewMaintainer(smCtx context.Context, repository storage.Repository, options *Settings) *Maintainer {
	return &Maintainer{
		smCtx:                   smCtx,
		repository:              repository,
		jobTimeout:              options.JobTimeout,
		operationExpirationTime: options.ExpirationTime,
		markOrphansInterval:     options.MarkOrphansInterval,
		cleanupInterval:         options.CleanupInterval,
	}
}

// Run starts the two recurring jobs responsible for cleaning up operations which are too old
// and deleting orphan operations
func (om *Maintainer) Run() {
	// TODO: Should all maintainer funcs be run initially?
	om.cleanupExternalOperations()
	om.cleanupInternalSuccessfulOperations()
	om.cleanupInternalFailedOperations()
	om.markOrphanOperationsFailed()

	go om.processOperations(om.cleanupExternalOperations, om.cleanupInterval)
	go om.processOperations(om.cleanupInternalSuccessfulOperations, om.cleanupInterval)
	go om.processOperations(om.cleanupInternalFailedOperations, 2*om.cleanupInterval)
	go om.processOperations(om.markOrphanOperationsFailed, om.markOrphansInterval)
}

// TODO: Consider some kind of Maintainer Func abstraction
func (om *Maintainer) processOperations(maintainerFunc func(), interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			maintainerFunc()
		case <-om.smCtx.Done():
			ticker.Stop()
			log.C(om.smCtx).Info("Server is shutting down. Stopping operations maintainer...")
			return
		}
	}
}

// TODO: fix (specialize) maintainer func logs
// TODO: Leave out the last operation for each resource id

// cleanUpExternalOperations cleans up periodically all external operations which are older than some specified time
func (om *Maintainer) cleanupExternalOperations() {
	criteria := []query.Criterion{
		query.ByField(query.NotEqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.operationExpirationTime))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up operations")
}

// cleanupInternalSuccessfulOperations cleans up periodically all successful internal operations which are older than some specified time
func (om *Maintainer) cleanupInternalSuccessfulOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.SUCCEEDED)),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.operationExpirationTime))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up operations")
}

// cleanupInternalFailedOperations cleans up periodically all failed internal operations which are older than some specified time
func (om *Maintainer) cleanupInternalFailedOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "state", string(types.FAILED)),
		query.ByField(query.EqualsOperator, "reschedule", "false"),
		query.ByField(query.EqualsOperator, "deletion_scheduled", "0001-01-01 00:00:00+00"),
		query.ByField(query.LessThanOperator, "updated_at", util.ToRFCNanoFormat(time.Now().Add(-om.operationExpirationTime))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up operations")
}

// markOrphanOperationsFailed periodically checks for operations which are stuck in state IN_PROGRESS and updates their status to FAILED
func (om *Maintainer) markOrphanOperationsFailed() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.LessThanOperator, "created_at", util.ToRFCNanoFormat(time.Now().Add(-om.jobTimeout))),
	}

	objectList, err := om.repository.List(om.smCtx, types.OperationType, criteria...)
	if err != nil {
		log.D().Debugf("Failed to fetch orphan operations: %s", err)
		return
	}

	operations := objectList.(*types.Operations)
	for i := 0; i < operations.Len(); i++ {
		operation := operations.ItemAt(i).(*types.Operation)
		operation.State = types.FAILED

		if _, err := om.repository.Update(om.smCtx, operation, query.LabelChanges{}); err != nil {
			log.D().Debugf("Failed to update orphan operation with ID (%s) state to FAILED: %s", operation.ID, err)
		}
	}

	log.D().Debug("Successfully marked orphan operations as failed")
}
