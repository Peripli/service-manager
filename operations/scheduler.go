package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

const scheduleMsg = "Scheduling %s job for operation with id (%s)"

// DefaultScheduler implements JobScheduler interface. It's responsible for
// storing C/U/D jobs so that a worker pool can eventually start consuming these jobs
type DefaultScheduler struct {
	repository storage.Repository
	workerPool *WorkerPool
	jobs       chan ExecutableJob
}

// NewScheduler constructs a DefaultScheduler
func NewScheduler(repository storage.Repository, workerPool *WorkerPool) *DefaultScheduler {
	return &DefaultScheduler{
		repository: repository,
		workerPool: workerPool,
		jobs:       workerPool.jobs,
	}
}

// Run starts the DefaultScheduler's worker pool enabling him to start polling
// for scheduled jobs
func (ds *DefaultScheduler) Run() {
	ds.workerPool.Run()
}

// Schedule schedules a CREATE/UPDATE/DELETE job in the worker pool
func (ds *DefaultScheduler) Schedule(reqCtx context.Context, objectType types.ObjectType, operation *types.Operation, operationFunc func(ctx context.Context, repository storage.Repository) (types.Object, error)) {
	log.D().Infof(scheduleMsg, operation.Type, operation.ID)
	ds.jobs <- &Job{
		operation:     operation,
		operationFunc: operationFunc,
		objectType:    objectType,
		reqCtx:        reqCtx,
	}
}
