package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"
	"sync"
	"time"
)

// WorkerPool is an abstraction responsible for processing
// jobs which are scheduled by a JobScheduler
type WorkerPool struct {
	smCtx          context.Context
	repository     storage.Repository
	jobs           chan ExecutableJob
	jobTimeout     time.Duration
	poolSize       int
	currentWorkers int
	mutex          *sync.RWMutex
}

// NewWorkerPool constructs a new worker pool
func NewWorkerPool(ctx context.Context, repository storage.Repository, poolSize int) *WorkerPool {
	return &WorkerPool{
		smCtx:          ctx,
		repository:     repository,
		jobs:           make(chan ExecutableJob, poolSize),
		poolSize:       poolSize,
		mutex:          &sync.RWMutex{},
		currentWorkers: 0,
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
		for wp.currentWorkers >= wp.poolSize {
		}

		go func() {
			defer func() {
				wp.mutex.Lock()
				wp.currentWorkers--
				wp.mutex.Unlock()
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

		wp.mutex.Lock()
		wp.currentWorkers++
		wp.mutex.Unlock()
	}
}
