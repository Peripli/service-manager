package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"
	"sync"
	"time"
)

// WorkerPool is an abstraction responsible for processing
// jobs which are scheduleed by a JobScheduler
type WorkerPool struct {
	repository     storage.Repository
	jobs           chan ExecutableJob
	errors         chan error
	jobTimeout     time.Duration
	poolSize       int
	currentWorkers int
	mutex          *sync.RWMutex
}

// NewWorkerPool constructs a new worker pool
func NewWorkerPool(repository storage.Repository, poolSize int) *WorkerPool {
	return &WorkerPool{
		repository:     repository,
		jobs:           make(chan ExecutableJob, poolSize),
		errors:         make(chan error, poolSize),
		poolSize:       poolSize,
		currentWorkers: 0,
	}
}

// Run starts the worker pool so it can start polling for scheduled jobs
func (wp *WorkerPool) Run() {
	wp.proccess()
}

func (wp *WorkerPool) proccess() {
	go wp.proccessJobs()
	go wp.proccessErrors()
}

// proccessJobs polls the currently scheduled jobs and as long as there
// are available workers it assigns a worker to execute for each available scheduled job
func (wp *WorkerPool) proccessJobs() {
	for {
		job := <-wp.jobs
		for wp.currentWorkers >= wp.poolSize {
		}

		go func() {
			ctxWithTimeout, cancel := context.WithTimeout(context.Background(), wp.jobTimeout)
			defer cancel()

			job.Execute(ctxWithTimeout, wp.repository, wp.errors)
			// TODO: gracefully handle operation failure when timeout + modify currentWorkers
		}()

		wp.mutex.Lock()
		wp.currentWorkers++
		wp.mutex.Unlock()
	}
}

func (wp *WorkerPool) proccessErrors() {
	for err := range wp.errors {
		if err != nil {
			log.D().Debugf("Error executing operation: %s", err.Error())
		}

		wp.mutex.Lock()
		wp.currentWorkers--
		wp.mutex.Unlock()
	}
}
