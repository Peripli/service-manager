package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

const scheduleMsg = "Scheduling %s job for operation with id (%s)"

// DefaultScheduler implements JobScheduler interface. It's responsible for
// storing C/U/D jobs so that a worker pool can eventually start consuming these jobs
type DefaultScheduler struct {
	smCtx      context.Context
	repository storage.Repository
	workerPool *WorkerPool
	jobs       chan ExecutableJob
}

// NewScheduler constructs a DefaultScheduler
func NewScheduler(smCtx context.Context, repository storage.Repository, poolSize int) *DefaultScheduler {
	workerPool := NewWorkerPool(repository, poolSize)

	return &DefaultScheduler{
		smCtx:      smCtx,
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

// SchedulerCreate schedules a Create job in the worker pool
func (ds *DefaultScheduler) ScheduleCreate(ctx context.Context, object types.Object, operationID string) {
	childCtx, childCtxCancel := context.WithCancel(ctx)

	go func() {
		log.D().Infof(scheduleMsg, types.CREATE, operationID)
		ds.jobs <- &CreateJob{
			baseJob: baseJob{
				operationID:  operationID,
				reqCtx:       childCtx,
				reqCtxCancel: childCtxCancel,
			},
			object: object,
		}
	}()
}

// SchedulerUpdate schedules an Update job in the worker pool
func (ds *DefaultScheduler) ScheduleUpdate(ctx context.Context, object types.Object, labelChanges query.LabelChanges, criteria []query.Criterion, operationID string) {
	childCtx, childCtxCancel := context.WithCancel(ctx)

	go func() {
		log.D().Infof(scheduleMsg, types.UPDATE, operationID)
		ds.jobs <- &UpdateJob{
			baseJob: baseJob{
				operationID:  operationID,
				reqCtx:       childCtx,
				reqCtxCancel: childCtxCancel,
			},
			object:       object,
			labelChanges: labelChanges,
			criteria:     criteria,
		}
	}()
}

// SchedulerDelete schedules an Delete job in the worker pool
func (ds *DefaultScheduler) ScheduleDelete(ctx context.Context, objectType types.ObjectType, criteria []query.Criterion, operationID string) {
	childCtx, childCtxCancel := context.WithCancel(ctx)

	go func() {
		log.D().Infof(scheduleMsg, types.DELETE, operationID)
		ds.jobs <- &DeleteJob{
			baseJob: baseJob{
				operationID:  operationID,
				reqCtx:       childCtx,
				reqCtxCancel: childCtxCancel,
			},
			objectType: objectType,
			criteria:   criteria,
		}
	}()
}
