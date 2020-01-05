package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"net/http"
	"time"
)

// DefaultScheduler implements JobScheduler interface. It's responsible for
// storing C/U/D jobs so that a worker pool can eventually start consuming these jobs
type DefaultScheduler struct {
	smCtx      context.Context
	repository storage.Repository

	workers        chan struct{}
	workerPoolSize int

	jobTimeout time.Duration
}

// NewScheduler constructs a DefaultScheduler
func NewScheduler(smCtx context.Context, repository storage.Repository, jobTimeout time.Duration, workerPoolSize int) *DefaultScheduler {
	return &DefaultScheduler{
		smCtx:      smCtx,
		repository: repository,
		workers:    make(chan struct{}, workerPoolSize),
		jobTimeout: jobTimeout,
	}
}

// Schedule schedules a CREATE/UPDATE/DELETE job and executes it asynchronously when as soon as possible
func (ds *DefaultScheduler) Schedule(job Job) (string, error) {
	log.D().Infof("Scheduling %s operation with id (%s)", job.Operation.Type, job.Operation.ID)
	select {
	case ds.workers <- struct{}{}:
		log.D().Infof("Storing %s operation with id (%s)", job.Operation.Type, job.Operation.ID)
		if _, err := ds.repository.Create(job.ReqCtx, job.Operation); err != nil {
			<-ds.workers
			return "", util.HandleStorageError(err, job.Operation.GetType().String())
		}

		go func() {
			defer func() {
				<-ds.workers
			}()

			ctxWithTimeout, cancel := context.WithTimeout(ds.smCtx, ds.jobTimeout)
			defer cancel()

			operationID, err := job.Execute(ctxWithTimeout, ds.repository)
			if err != nil {
				log.D().Debugf("Error occurred during execution of operation with ID (%s): %s", operationID, err.Error())
				return
			}
			log.D().Debugf("Successful executed operation with ID (%s)", operationID)
		}()
	default:
		log.D().Infof("Failed to schedule %s operation with id (%s) - all workers are busy.", job.Operation.Type, job.Operation.ID)
		return "", &util.HTTPError{
			ErrorType:   "ServiceUnavailable",
			Description: "Failed to schedule operation. Server is busy - try again in a few minutes.",
			StatusCode:  http.StatusServiceUnavailable,
		}
	}

	return job.Operation.ID, nil
}
