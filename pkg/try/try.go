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

package try

import (
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sony/gobreaker"
)

var (
	breakers    = make(map[string]*gobreaker.CircuitBreaker)
	statesMutex = &sync.Mutex{}
)

var DefaultBreakerSettings = gobreaker.Settings{
	Timeout: 30 * time.Second,
	ReadyToTrip: func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= 3 && failureRatio >= 0.6
	},
}

type Action struct {
	off     backoff.BackOff
	breaker *gobreaker.CircuitBreaker
}

var DefaultAction = &Action{
	off:     backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3),
	breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{}),
}

func (a *Action) WithBackoff(off backoff.BackOff) *Action {
	a.off = off
	return a
}

func (a *Action) WithCircuit(settings gobreaker.Settings) *Action {
	a.breaker = gobreaker.NewCircuitBreaker(settings)
	return a
}

func (a *Action) Execute(operation func() error) error {
	_, err := a.breaker.Execute(func() (interface{}, error) {
		return nil, backoff.Retry(operation, a.off)
	})
	return err
}

func To(action string) *Action {
	statesMutex.Lock()
	defer statesMutex.Unlock()
	breaker, exists := breakers[action]
	if !exists {
		breakerSettings := gobreaker.Settings{
			Name:          action,
			Timeout:       DefaultBreakerSettings.Timeout,
			Interval:      DefaultBreakerSettings.Interval,
			ReadyToTrip:   DefaultBreakerSettings.ReadyToTrip,
			MaxRequests:   DefaultBreakerSettings.MaxRequests,
			OnStateChange: DefaultBreakerSettings.OnStateChange,
		}
		breaker = gobreaker.NewCircuitBreaker(breakerSettings)
		breakers[action] = breaker
	}

	return &Action{
		breaker: breaker,
		off:     DefaultAction.off,
	}
}
