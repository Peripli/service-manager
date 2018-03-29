/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package server

import "time"

type config struct {
	address         string
	requestTimeout  time.Duration
	shutdownTimeout time.Duration
}

func (config *config) Address() string {
	return config.address
}

func (config *config) RequestTimeout() time.Duration {
	return config.requestTimeout
}

func (config *config) ShutdownTimeout() time.Duration {
	return config.shutdownTimeout
}

// DefaultConfiguration returns a default server configuration
func DefaultConfiguration() Configuration {
	return &config{
		address:         ":8080",
		requestTimeout:  time.Millisecond * time.Duration(1500),
		shutdownTimeout: time.Second * time.Duration(5),
	}
}
