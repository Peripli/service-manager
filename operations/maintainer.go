package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"time"
)

// OperationMaintainer ensures that operations old enough are deleted
// and that no stuck (orphan) operations are left in the DB due to crashes/restarts of SM
type OperationMaintainer struct {
	repository      storage.Repository
	jobTimeout      time.Duration
	cleanupInterval time.Duration
}

// NewOperationMaintainer constructs an OperationMaintainer
func NewOperationMaintainer(repository storage.Repository, options *Settings) *OperationMaintainer {
	return &OperationMaintainer{
		repository:      repository,
		jobTimeout:      options.JobTimeout,
		cleanupInterval: options.CleanupInterval,
	}
}

// Run starts the two recurring jobs responsible for cleaning up old operations
// and deleting stuck (orphan) operations
func (om *OperationMaintainer) Run() {
	go om.cleanupOperations()
	go om.cleanupStuckOperations()
}

// cleanOperations cleans up periodically all operations but the last
// for each C/U/D operation for every resource_id which are older than some specified time
func (om *OperationMaintainer) cleanupOperations() {
	ticker := time.NewTicker(om.cleanupInterval)
	terminate := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			om.deleteOldOperations()
		case <-terminate:
			ticker.Stop()
			return
		}
	}
}

// cleanupStuckOperations periodically cleans up all operations which are stuck in state IN_PROGRESS
func (om *OperationMaintainer) cleanupStuckOperations() {
	ticker := time.NewTicker(om.jobTimeout)
	terminate := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			om.deleteOrphanOperations()
		case <-terminate:
			ticker.Stop()
			return
		}
	}
}

func (om *OperationMaintainer) deleteOldOperations() {
	// TODO: don't cleanup all until two hours ago, clean up all but the last C/U/D opeartion for each resource_id
	byDate := query.ByField(query.LessThanOperator, "created_at", time.Now().Add(-om.cleanupInterval).String())
	if err := om.repository.Delete(context.Background(), types.OperationType, byDate); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up operations")
}

func (om *OperationMaintainer) deleteOrphanOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.LessThanOperator, "created_at", time.Now().Add(-om.jobTimeout).String()),
	}

	if err := om.repository.Delete(context.Background(), types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup orphan operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up orphan operations")
}
