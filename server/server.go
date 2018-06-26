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
	"net/http"
	"time"

	"fmt"

	"strconv"

	"github.com/Peripli/service-manager/rest"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Settings type to be loaded from the environment
type Settings struct {
	Host            string
	Port            int
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// Server is the server to process incoming HTTP requests
type Server struct {
	Config Settings
	Router *mux.Router
}

// New creates a new server with the provided REST API configuration and server configuration
// Returns the new server and an error if creation was not successful
func New(api rest.API, config Settings) (*Server, error) {
	router := mux.NewRouter().StrictSlash(true)
	if err := registerControllers(router, api.Controllers()); err != nil {
		return nil, fmt.Errorf("new Settings: %s", err)
	}

	return &Server{
		Config: config,
		Router: router,
	}, nil
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

func registerRoutes(prefix string, fromRouter *mux.Router, toRouter *mux.Router) error {
	subRouter := toRouter.PathPrefix(prefix).Subrouter()
	return fromRouter.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {

		path, err := route.GetPathTemplate()
		if err != nil {
			return fmt.Errorf("register routes: %s", err)
		}
		r := subRouter.Handle(path, route.GetHandler())

		methods, err := route.GetMethods()
		if err != nil {
			return fmt.Errorf("register routes: %s", err)

		}
		if len(methods) > 0 {
			r.Methods(methods...)
		}

		logrus.Debug("Registering route: method: ", methods, " path: ", prefix, path)
		return nil
	})
}

func registerControllers(router *mux.Router, controllers []rest.Controller) error {
	for _, ctrl := range controllers {
		for _, route := range ctrl.Routes() {
			fromRouter, ok := route.Handler.(*mux.Router)
			if ok {
				if err := registerRoutes(route.Endpoint.Path, fromRouter, router); err != nil {

					return fmt.Errorf("register controllers: %s", err)
				}
			} else {
				r := router.Handle(route.Endpoint.Path, route.Handler)
				if route.Endpoint.Method != rest.AllMethods {
					r.Methods(route.Endpoint.Method)
				}
			}
		}
	}
	return nil
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
