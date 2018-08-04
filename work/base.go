/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package work

import "fmt"

type EntityType int

const (
	EntityBroker EntityType = iota
	EntityPlatform
)

type Action int

const (
	ActionCreate Action = iota
	ActionDelete
	ActionUpdate
)

var ActionNames map[Action]string

func init() {
	ActionNames = make(map[Action]string)
	ActionNames[ActionCreate] = "Create"
	ActionNames[ActionDelete] = "Delete"
	ActionNames[ActionUpdate] = "Update"
}

type Worker interface {
	Supports(job Job) bool
	Work(job Job) error
}

type Job struct {
	EntityType EntityType
	Action     Action
	EntityId   string
	Data       []byte
}

type Master struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
	delegates  []Worker
}

func NewMaster(workerPool chan chan Job, delegates []Worker) Master {
	return Master{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
		delegates:  delegates,
	}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Master) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel
			select {
			case job := <-w.JobChannel:
				// we have received a work request.
				w.process(job)
			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}

// Stop signals the worker to stop listening for work requests.
func (w Master) Stop() {
	go func() {
		w.quit <- true
	}()
}

func (w Master) process(job Job) {
	for i := range w.delegates {
		delegate := w.delegates[i]
		if delegate.Supports(job) {
			if err := delegate.Work(job); err != nil {
				fmt.Printf("Error: %v", err)
			}
		}
	}
}
