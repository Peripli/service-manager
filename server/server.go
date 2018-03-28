/*
 *    Copyright 2018 The Service Manager Authors
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

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Peripli/service-manager/rest"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// Configuration represents the configuration to use for the server
type Configuration interface {
	// Address returns the port on which the server to listen. The port must be preceded by a colon, for example ":8080"
	Address() string

	// RequestTimeout returns the read/write request timeout
	RequestTimeout() time.Duration

	// ShutdownTimeout returns the timeout for which the server to wait running tasks before terminating
	ShutdownTimeout() time.Duration
}

// Server is the server to process incoming HTTP requests
type Server struct {
	Configuration Configuration
	Router        *mux.Router
}

// New creates a new server with the provided REST API configuration and server configuration
// Returns the new server and an error if creation was not successful
func New(api rest.Api, config Configuration) (*Server, error) {
	router := mux.NewRouter().StrictSlash(true)
	registerControllers(router, api.Controllers())

	return &Server{config, router}, nil
}

// Run starts the server awaiting for incoming requests
func (server *Server) Run() {
	handler := &http.Server{
		Handler:      server.Router,
		Addr:         server.Configuration.Address(),
		WriteTimeout: server.Configuration.RequestTimeout(),
		ReadTimeout:  server.Configuration.RequestTimeout(),
	}
	startServer(handler, server.Configuration.ShutdownTimeout())
}

func registerControllers(router *mux.Router, controllers []rest.Controller) {
	for _, ctrl := range controllers {
		for _, route := range ctrl.Routes() {
			router.Handle(route.Endpoint.Path, rest.ErrorHandlerFunc(route.Handler)).Methods(route.Endpoint.Method)
		}
	}
}

func startServer(server *http.Server, shutdownTimeout time.Duration) {
	go gracefulShutdown(server, shutdownTimeout)

	logrus.Debugf("Listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatal(err)
	}
}

func gracefulShutdown(server *http.Server, shutdownTimeout time.Duration) {
	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logrus.Debugf("Shutdown with timeout: %s", shutdownTimeout)

	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("Error: %v", err)
	} else {
		logrus.Debug("Server stopped")
	}
}
