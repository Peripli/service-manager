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

// SchedulerCreate schedules a Create job in the worker pool
func (ds *DefaultScheduler) ScheduleCreate(reqCtx context.Context, reqCtxCancel context.CancelFunc, object types.Object, operationID string) {
	log.D().Infof(scheduleMsg, types.CREATE, operationID)
	ds.jobs <- &CreateJob{
		baseJob: baseJob{
			operationID:  operationID,
			reqCtx:       reqCtx,
			reqCtxCancel: reqCtxCancel,
		},
		object: object,
	}
}

// SchedulerUpdate schedules an Update job in the worker pool
func (ds *DefaultScheduler) ScheduleUpdate(reqCtx context.Context, reqCtxCancel context.CancelFunc, object types.Object, labelChanges query.LabelChanges, criteria []query.Criterion, operationID string) {
	log.D().Infof(scheduleMsg, types.UPDATE, operationID)
	ds.jobs <- &UpdateJob{
		baseJob: baseJob{
			operationID:  operationID,
			reqCtx:       reqCtx,
			reqCtxCancel: reqCtxCancel,
		},
		object:       object,
		labelChanges: labelChanges,
		criteria:     criteria,
	}
}

// SchedulerDelete schedules an Delete job in the worker pool
func (ds *DefaultScheduler) ScheduleDelete(reqCtx context.Context, reqCtxCancel context.CancelFunc, objectType types.ObjectType, criteria []query.Criterion, operationID string) {
	log.D().Infof(scheduleMsg, types.DELETE, operationID)
	ds.jobs <- &DeleteJob{
		baseJob: baseJob{
			operationID:  operationID,
			reqCtx:       reqCtx,
			reqCtxCancel: reqCtxCancel,
		},
		objectType: objectType,
		criteria:   criteria,
	}
}
