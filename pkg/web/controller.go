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

package web

// Controller is an entity that wraps a set of HTTP Routes
type Controller interface {
	// Routes returns the set of routes for this controller
	Routes() []Route
}

// Route is a mapping between an Endpoint and a REST API SMHandler
type Route struct {
	// Endpoint is the combination of Path and HTTP Method for the specified route
	Endpoint Endpoint

	// Handler is the function that should handle incoming requests for this endpoint
	Handler HandlerFunc
}

// Endpoint is a combination of a Path and an HTTP Method
type Endpoint struct {
	Method string
	Path   string
}
