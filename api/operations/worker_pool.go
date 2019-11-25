package operations

import (
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"
)

type WorkerPool struct {
	repository     storage.Repository
	jobs           chan ExecutableJob
	errors         chan error
	poolSize       int
	currentWorkers int
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
	go wp.proccessJobs()
	go wp.proccessErrors()
}

func (wp *WorkerPool) proccessJobs() {
	for {
		job := <-wp.jobs
		for wp.currentWorkers >= wp.poolSize {
		}
		go job.Execute(wp.repository, wp.errors)
		wp.currentWorkers++
	}
}

func (wp *WorkerPool) proccessErrors() {
	for err := range wp.errors {
		if err != nil {
			log.D().Debugf("Error executing operation: %s", err.Error())
		}
		wp.currentWorkers--
	}
}
