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
	"time"

	"github.com/Peripli/service-manager/rest"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Server is the server to process incoming HTTP requests
type Server struct {
	Configuration *config
	Router        *mux.Router
}

// New creates a new server with the provided REST API configuration and server configuration
// Returns the new server and an error if creation was not successful
func New(api rest.API, config *config) (*Server, error) {
	router := mux.NewRouter().StrictSlash(true)
	registerControllers(router, api.Controllers())

	setUpLogging(config.LogLevel, config.LogFormat)

	return &Server{
		Configuration: config,
		Router:        router,
	}, nil
}

// Run starts the server awaiting for incoming requests
func (server *Server) Run(ctx context.Context) {
	handler := &http.Server{
		Handler:      server.Router,
		Addr:         server.Configuration.Address,
		WriteTimeout: server.Configuration.RequestTimeout,
		ReadTimeout:  server.Configuration.RequestTimeout,
	}
	startServer(handler, server.Configuration.ShutdownTimeout, ctx)
}

func moveRoutes(prefix string, fromRouter *mux.Router, toRouter *mux.Router) error {
	subRouter := toRouter.PathPrefix(prefix).Subrouter()
	return fromRouter.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {

		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}

		methods, err := route.GetMethods()
		if err != nil {
			return err
		}

		logrus.Info("Adding route with methods: ", methods, " and path: ", path)
		subRouter.Handle(path, route.GetHandler()).Methods(methods...)
		return nil
	})
}

func registerControllers(router *mux.Router, controllers []rest.Controller) {
	for _, ctrl := range controllers {
		for _, route := range ctrl.Routes() {
			fromRouter, ok := route.Handler.(*mux.Router)
			if ok {
				moveRoutes(route.Endpoint.Path, fromRouter, router)
			} else {
				r := router.Handle(route.Endpoint.Path, route.Handler)
				if route.Endpoint.Method != rest.AllMethods {
					r.Methods(route.Endpoint.Method)
				}
			}
		}
	}
}

func startServer(server *http.Server, shutdownTimeout time.Duration, ctx context.Context) {
	go gracefulShutdown(server, shutdownTimeout, ctx)

	logrus.Debugf("Listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatal(err)
	}
}

func gracefulShutdown(server *http.Server, shutdownTimeout time.Duration, ctx context.Context) {
	<-ctx.Done()

	c, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	logrus.Debugf("Shutdown with timeout: %s", shutdownTimeout)

	if err := server.Shutdown(c); err != nil {
		server.Close()
		logrus.Errorf("Error: %v", err)
	} else {
		logrus.Debug("Server stopped")
	}
}

func setUpLogging(logLevel string, logFormat string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatal("Could not parse log level configuration")
	}
	logrus.SetLevel(level)
	if logFormat == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
}
