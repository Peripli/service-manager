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

// Package server contains the logic of the Service Manager server
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"strconv"

	"github.com/Peripli/service-manager/rest"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Settings type to be loaded from the environment
type Settings struct {
	Port            int
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// Validate validates the server settings
func (s *Settings) Validate() error {
	if s.Port == 0 {
		return fmt.Errorf("validate Settings: Port missing")
	}
	if s.RequestTimeout == 0 {
		return fmt.Errorf("validate Settings: RequestTimeout missing")
	}
	if s.ShutdownTimeout == 0 {
		return fmt.Errorf("validate Settings: ShutdownTimeout missing")
	}
	return nil
}

// Server is the server to process incoming HTTP requests
type Server struct {
	Config  Settings
	Handler http.Handler
}

// New creates a new server with the provided REST API configuration and server configuration
// Returns the new server and an error if creation was not successful
func New(api *rest.API, config Settings) *Server {
	router := mux.NewRouter().StrictSlash(true)
	registerControllers(router, api)

	recoveryHandler := handlers.RecoveryHandler(
		handlers.PrintRecoveryStack(true),
		handlers.RecoveryLogger(&recoveryHandlerLogger{}),
	)(router)

	return &Server{
		Config:  config,
		Handler: recoveryHandler,
	}
}

type recoveryHandlerLogger struct{}

// PrintLn prints panic message and stack to error output
func (r *recoveryHandlerLogger) Println(args ...interface{}) {
	logrus.Errorln(args...)
}

// Run starts the server awaiting for incoming requests
func (s *Server) Run(ctx context.Context) {
	handler := &http.Server{
		Handler:      s.Handler,
		Addr:         ":" + strconv.Itoa(s.Config.Port),
		WriteTimeout: s.Config.RequestTimeout,
		ReadTimeout:  s.Config.RequestTimeout,
	}
	startServer(ctx, handler, s.Config.ShutdownTimeout)
}

func registerControllers(router *mux.Router, api *rest.API) {
	for _, ctrl := range api.Controllers {
		for _, route := range ctrl.Routes() {
			logrus.Debugf("Register endpoint: %s %s", route.Endpoint.Method, route.Endpoint.Path)
			filters := rest.MatchFilters(&route.Endpoint, api.Filters)
			handler := rest.NewHTTPHandler(filters, route.Handler)
			r := router.Handle(route.Endpoint.Path, handler)
			r.Methods(route.Endpoint.Method)
		}
	}
}

func startServer(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) {
	go gracefulShutdown(ctx, server, shutdownTimeout)

	logrus.Infof("Listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatal(err)
	}
}

func gracefulShutdown(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) {
	<-ctx.Done()

	c, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	logrus.Debugf("Shutdown with timeout: %s", shutdownTimeout)

	if err := server.Shutdown(c); err != nil {
		logrus.Error("Error: ", err)
		if err := server.Close(); err != nil {
			logrus.Error("Error: ", err)
		}
	} else {
		logrus.Debug("Server stopped")
	}
}
