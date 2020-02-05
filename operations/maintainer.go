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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"time"
)

// Maintainer ensures that operations old enough are deleted
// and that no orphan operations are left in the DB due to crashes/restarts of SM
type Maintainer struct {
	smCtx                   context.Context
	repository              storage.Repository
	jobTimeout              time.Duration
	markOrphansInterval     time.Duration
	cleanupInterval         time.Duration
	operationExpirationTime time.Duration
}

// NewMaintainer constructs a Maintainer
func NewMaintainer(smCtx context.Context, repository storage.Repository, options *Settings) *Maintainer {
	return &Maintainer{
		smCtx:                   smCtx,
		repository:              repository,
		jobTimeout:              options.JobTimeout,
		markOrphansInterval:     options.MarkOrphansInterval,
		cleanupInterval:         options.CleanupInterval,
		operationExpirationTime: options.ExpirationTime,
	}
}

// Run starts the two recurring jobs responsible for cleaning up operations which are too old
// and deleting orphan operations
func (om *Maintainer) Run() {
	om.cleanUpOldOperations()
	om.markOrphanOperationsFailed()

	go om.processOldOperations()
	go om.processOrphanOperations()
}

// processOldOperations cleans up periodically all operations which are older than some specified time
func (om *Maintainer) processOldOperations() {
	ticker := time.NewTicker(om.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			om.cleanUpOldOperations()
		case <-om.smCtx.Done():
			ticker.Stop()
			log.C(om.smCtx).Info("Server is shutting down. Stopping old operations maintainer...")
			return
		}
	}
}

// processOrphanOperations periodically checks for operations which are stuck in state IN_PROGRESS and updates their status to FAILED
func (om *Maintainer) processOrphanOperations() {
	ticker := time.NewTicker(om.markOrphansInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			om.markOrphanOperationsFailed()
		case <-om.smCtx.Done():
			ticker.Stop()
			log.C(om.smCtx).Info("Server is shutting down. Stopping stuck operations maintainer...")
			return
		}
	}
}

func (om *Maintainer) cleanUpOldOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "state", string(types.FAILED)),
		query.ByField(query.NotEqualsOperator, "platform_id", types.SMPlatform),
		query.ByField(query.LessThanOperator, "created_at", util.ToRFCNanoFormat(time.Now().Add(-om.operationExpirationTime))),
	}

	if err := om.repository.Delete(om.smCtx, types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up operations")
}

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
