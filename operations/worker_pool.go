package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"
	"time"
)

// WorkerPool is an abstraction responsible for processing
// jobs which are scheduled by a JobScheduler
type WorkerPool struct {
	smCtx      context.Context
	repository storage.Repository
	jobs       chan ExecutableJob
	workers    chan struct{}
	jobTimeout time.Duration
}

// NewWorkerPool constructs a new worker pool
func NewWorkerPool(ctx context.Context, repository storage.Repository, options *Settings) *WorkerPool {
	return &WorkerPool{
		smCtx:      ctx,
		repository: repository,
		jobs:       make(chan ExecutableJob, options.PoolSize),
		workers:    make(chan struct{}, options.PoolSize),
		jobTimeout: options.JobTimeout,
	}
}

// Run starts the worker pool so it can start polling for scheduled jobs
func (wp *WorkerPool) Run() {
	go wp.processJobs()
}

// processJobs polls the currently scheduled jobs and as long as there
// are available workers it assigns a worker to execute for each available scheduled job
func (wp *WorkerPool) processJobs() {
	for {
		job := <-wp.jobs
		wp.workers <- struct{}{}

		go func() {
			defer func() {
				<-wp.workers
			}()

			ctxWithTimeout, cancel := context.WithTimeout(wp.smCtx, wp.jobTimeout)
			defer cancel()

			operationID, err := job.Execute(ctxWithTimeout, wp.repository)
			if err != nil {
				log.D().Debugf("Error occurred during execution of operation with ID (%s): %s", operationID, err.Error())
			} else {
				log.D().Debugf("Successful executed operation with ID (%s)", operationID)
			}
		}()
	}
}
