package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"time"
)

const scheduleMsg = "Scheduling %s job for operation with id (%s)"

type DefaultScheduler struct {
	smCtx      context.Context
	repository storage.Repository
	workerPool *WorkerPool
	jobs       chan ExecutableJob
}

func NewScheduler(smCtx context.Context, repository storage.Repository, poolSize int, jobTimeout time.Duration) *DefaultScheduler {
	workerPool := NewPool(repository, poolSize)

	return &DefaultScheduler{
		smCtx:      smCtx,
		repository: repository,
		workerPool: workerPool,
		jobs:       workerPool.jobs,
	}
}

func (ds *DefaultScheduler) Run() {
	ds.workerPool.Run()
}

func (ds *DefaultScheduler) ScheduleCreate(ctx context.Context, object types.Object, operationID string) {
	go func() {
		log.D().Infof(scheduleMsg, types.CREATE, operationID)
		ds.jobs <- &CreateJob{
			operationID: operationID,
			reqCtx:      ctx,
			object:      object,
		}
	}()
}

func (ds *DefaultScheduler) ScheduleUpdate(ctx context.Context, object types.Object, labelChanges query.LabelChanges, criteria []query.Criterion, operationID string) {
	go func() {
		log.D().Infof(scheduleMsg, types.UPDATE, operationID)
		ds.jobs <- &UpdateJob{
			operationID:  operationID,
			reqCtx:       ctx,
			object:       object,
			labelChanges: labelChanges,
			criteria:     criteria,
		}
	}()
}

func (ds *DefaultScheduler) ScheduleDelete(ctx context.Context, objectType types.ObjectType, criteria []query.Criterion, operationID string) {
	go func() {
		log.D().Infof(scheduleMsg, types.DELETE, operationID)
		ds.jobs <- &DeleteJob{
			operationID: operationID,
			reqCtx:      ctx,
			objectType:  objectType,
			criteria:    criteria,
		}
	}()
}
