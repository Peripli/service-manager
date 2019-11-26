package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"
	"sync"
	"time"
)

type WorkerPool struct {
	repository     storage.Repository
	jobs           chan ExecutableJob
	errors         chan error
	jobTimeout     time.Duration
	poolSize       int
	currentWorkers int
	mutex          *sync.RWMutex
}

func NewPool(repository storage.Repository, poolSize int) *WorkerPool {
	return &WorkerPool{
		repository:     repository,
		jobs:           make(chan ExecutableJob, poolSize),
		errors:         make(chan error, poolSize),
		poolSize:       poolSize,
		currentWorkers: 0,
	}
}

func (wp *WorkerPool) Run() {
	wp.proccess()
}

func (wp *WorkerPool) proccess() {
	go wp.proccessJobs()
	go wp.proccessErrors()
}

func (wp *WorkerPool) proccessJobs() {
	for {
		job := <-wp.jobs
		for wp.currentWorkers >= wp.poolSize {
		}

		go func() {
			ctxWithTimeout, _ := context.WithTimeout(context.Background(), wp.jobTimeout)
			go job.Execute(ctxWithTimeout, wp.repository, wp.errors)
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
