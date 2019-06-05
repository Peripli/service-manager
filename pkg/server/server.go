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
	"sync"
	"time"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gorilla/mux"
)

const (
	_      = iota
	kb int = 1 << (10 * iota)
	mb
)

// Settings type to be loaded from the environment
type Settings struct {
	Host            string        `mapstructure:"host" description:"host of the server"`
	Port            int           `mapstructure:"port" description:"port of the server"`
	RequestTimeout  time.Duration `mapstructure:"request_timeout" description:"read and write timeout duration for requests"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" description:"time to wait for the server to shutdown"`
	MaxBodyBytes    int           `mapstructure:"max_body_bytes" description:"maximum bytes size of incoming body"`
	MaxHeaderBytes  int           `mapstructure:"max_header_bytes" description:"the maximum number of bytes the server will read parsing the request header"`
}

// DefaultSettings returns the default values for configuring the Service Manager
func DefaultSettings() *Settings {
	return &Settings{
		Port:            8080,
		RequestTimeout:  time.Second * 3,
		ShutdownTimeout: time.Second * 3,
		MaxBodyBytes:    mb,
		MaxHeaderBytes:  kb,
	}
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
	*mux.Router

	Config *Settings
}

// New creates a new server with the provided REST api configuration and server configuration
// Returns the new server and an error if creation was not successful
func New(config *Settings, api *web.API) *Server {
	router := mux.NewRouter().StrictSlash(true)
	registerControllers(api, router, config)

	return &Server{
		Router: router,
		Config: config,
	}
}

func registerControllers(API *web.API, router *mux.Router, config *Settings) {
	for _, ctrl := range API.Controllers {
		for _, route := range ctrl.Routes() {
			log.D().Debugf("Registering endpoint: %s %s", route.Endpoint.Method, route.Endpoint.Path)
			handler := web.Filters(API.Filters).ChainMatching(route)
			router.Handle(route.Endpoint.Path, api.NewHTTPHandler(handler, config.MaxBodyBytes)).Methods(route.Endpoint.Method)
		}
	}
}

// Run starts the server awaiting for incoming requests
func (s *Server) Run(ctx context.Context, wg *sync.WaitGroup) {
	if err := s.Config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid server config: %s", err))
	}
	handler := &http.Server{
		Handler:        s.Router,
		Addr:           s.Config.Host + ":" + strconv.Itoa(s.Config.Port),
		WriteTimeout:   s.Config.RequestTimeout,
		ReadTimeout:    s.Config.RequestTimeout,
		MaxHeaderBytes: s.Config.MaxHeaderBytes,
	}
	startServer(ctx, handler, s.Config.ShutdownTimeout, wg)
}

func startServer(ctx context.Context, server *http.Server, shutdownTimeout time.Duration, wg *sync.WaitGroup) {
	wg.Add(1)
	go gracefulShutdown(ctx, server, shutdownTimeout, wg)

	log.C(ctx).Infof("Server listening on %s...", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.C(ctx).Fatal(err)
	}
}

func gracefulShutdown(ctx context.Context, server *http.Server, shutdownTimeout time.Duration, wg *sync.WaitGroup) {
	<-ctx.Done()
	defer wg.Done()

	c, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	logger := log.C(ctx)
	logger.Debugf("Shutdown with timeout: %s", shutdownTimeout)

	if err := server.Shutdown(c); err != nil {
		logger.Error("Error: ", err)
		if err := server.Close(); err != nil {
			logger.Error("Error: ", err)
		}
	} else {
		logger.Debug("Server stopped")
	}
}
