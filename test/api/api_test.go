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
package api_itest

import (
	"context"

	"net/http/httptest"
	"os"
	"os/signal"
	"testing"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

var sm *httpexpect.Expect

func TestAPI(t *testing.T) {
	RunSpecs(t, "API Tests Suite")
}

// TODO: deduplicate with main.go
func handleInterrupts(ctx context.Context, cancelFunc context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Error("Received OS interrupt, exiting gracefully...")
			cancelFunc()
		case <-ctx.Done():
			return
		}
	}()
}

func getServerRouter() (*mux.Router, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	config, err := server.NewConfiguration(env.Default())
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
		return nil, err
	}

	if err := config.Validate(); err != nil {
		logrus.Fatal("NewConfiguration: validation failed ", err)
		return nil, err
	}

	storage, err := storage.Use(ctx, postgres.Storage, config.DbURI)
	if err != nil {
		logrus.Fatal("Error using storage: ", err)
		return nil, err
	}
	defaultAPI := api.Default(storage)

	srv, err := server.New(defaultAPI, config)
	if err != nil {
		logrus.Fatal("Error creating server: ", err)
		return nil, err
	}
	return srv.Router, nil
}

var _ = Describe("Service Manager API", func() {
	var testServer *httptest.Server

	BeforeSuite(func() {
		router, err := getServerRouter()
		if err != nil {
			panic(err)
		}
		testServer = httptest.NewServer(router)
		sm = httpexpect.New(GinkgoT(), testServer.URL)
	})

	AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("Service Brokers", testBrokers)

	Describe("Platforms", testPlatforms)
})
