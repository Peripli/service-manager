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

// Package server contains the logic of the Service Manager server and a mux router
package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Settings type to be loaded from the environment
type Settings struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	RequestTimeout  time.Duration `mapstructure:"request_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
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
	Config Settings
	Router *mux.Router
	API    *web.API
}

// New creates a new server with the provided REST API configuration and server configuration
// Returns the new server and an error if creation was not successful
func New(config Settings, api *web.API) *Server {
	router := mux.NewRouter().StrictSlash(true)
	registerControllers(api, router)

	return &Server{
		Config: config,
		Router: router,
		API:    api,
	}
}

// Run starts the server awaiting for incoming requests
func (s *Server) Run(ctx context.Context) {
	handler := &http.Server{
		Handler:      s.Router,
		Addr:         s.Config.Host + ":" + strconv.Itoa(s.Config.Port),
		WriteTimeout: s.Config.RequestTimeout,
		ReadTimeout:  s.Config.RequestTimeout,
	}
	startServer(ctx, handler, s.Config.ShutdownTimeout)
}

func registerControllers(API *web.API, router *mux.Router) {
	for _, ctrl := range API.Controllers {
		for _, route := range ctrl.Routes() {
			logrus.Debugf("Registering endpoint: %s %s", route.Endpoint.Method, route.Endpoint.Path)
			handler := web.Filters(API.Filters).ChainMatching(route)
			router.Handle(route.Endpoint.Path, api.NewHTTPHandler(handler)).Methods(route.Endpoint.Method)
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
