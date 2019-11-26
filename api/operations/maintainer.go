package operations

import (
	"context"
	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"time"
)

type OperationMaintainer struct {
	repository       storage.Repository
	jobTimeout       time.Duration
	cleanupInterval  time.Duration
	cleanupThreshold time.Duration
}

func NewOperationMaintainer(options *api.Options) *OperationMaintainer {
	return &OperationMaintainer{
		repository:       options.Repository,
		jobTimeout:       options.APISettings.JobTimeout,
		cleanupInterval:  options.APISettings.JobTimeout,
		cleanupThreshold: options.APISettings.JobTimeout,
	}
}

func (om *OperationMaintainer) Run() {
	go om.cleanupOperations()
	go om.cleanupOrphans()
}

// cleanOperations cleans up periodicallyall operations but the last
// for each C/U/D operation for every resource_id
func (om *OperationMaintainer) cleanupOperations() {
	ticker := time.NewTicker(om.cleanupInterval)
	terminate := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			om.deleteResourceOperations()
		case <-terminate:
			ticker.Stop()
			return
		}
	}
}

// cleanupOrphans periodically cleans up all operations which are stuck in state IN_PROGRESS
func (om *OperationMaintainer) cleanupOrphans() {
	ticker := time.NewTicker(om.jobTimeout * 2)
	terminate := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			om.deleteOrphanResourceOperations()
		case <-terminate:
			ticker.Stop()
			return
		}
	}
}

func (om *OperationMaintainer) deleteResourceOperations() {
	// TODO: don't cleanup all until two hours ago, clean up all but the last C/U/D opeartion for each resource_id
	byDate := query.ByField(query.LessThanOperator, "created_at", time.Now().Add(-om.cleanupThreshold).String())
	if _, err := om.repository.Delete(context.Background(), types.OperationType, byDate); err != nil {
		log.D().Debugf("Failed to cleanup operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up operations")
}

func (om *OperationMaintainer) deleteOrphanResourceOperations() {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.LessThanOperator, "created_at", time.Now().Add(-om.jobTimeout).String()),
	}

	if _, err := om.repository.Delete(context.Background(), types.OperationType, criteria...); err != nil {
		log.D().Debugf("Failed to cleanup orphan operations: %s", err)
		return
	}
	log.D().Debug("Successfully cleaned up orphan operations")
}
