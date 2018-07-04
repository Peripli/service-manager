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
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/Peripli/service-manager/rest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("Server", func() {

	Describe("New", func() {
		Context("when controller has route with no path", func() {
			It("should return error", func() {
				customRouter := mux.NewRouter().StrictSlash(true)
				customRouter.NewRoute().HandlerFunc(emptyHandler).Methods(http.MethodGet)
				_, err := newServer(customRouter, 0)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("route doesn't have a path"))
			})
		})

		Context("when controller has route with path", func() {
			It("should return server", func() {
				customRouter := mux.NewRouter().StrictSlash(true)
				customRouter.NewRoute().HandlerFunc(emptyHandler).Path("/").Methods(http.MethodGet)
				server, err := newServer(customRouter, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(server).ToNot(BeNil())
			})
		})

		Context("when controller has custom http.handler", func() {
			It("should return server", func() {
				server, err := newServer(&testHTTPHandler{}, 0)
				Expect(err).ToNot(HaveOccurred())
				Expect(server).ToNot(BeNil())
			})
		})
	})

	Describe("Run", func() {
		Context("With panicing handler", func() {
			It("should return status 500", func() {
				port := getFreePort()
				server, err := newServer(&testHTTPHandler{}, port)
				Expect(err).ToNot(HaveOccurred())
				ctx, cancel := context.WithCancel(context.Background())
				go func() { server.Run(ctx) }()
				defer cancel()
				baseURL := "http://localhost:" + strconv.Itoa(port)
				makeRequest(baseURL, http.StatusOK)
				makeRequest(baseURL+"?fail=true", http.StatusInternalServerError)
			})
		})
	})

})

func makeRequest(url string, expectedStatus int) {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	Expect(resp.StatusCode).To(Equal(expectedStatus))
}

func getFreePort() int {
	socket, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	port := socket.Addr().(*net.TCPAddr).Port
	socket.Close()
	return port
}

func emptyHandler(rw http.ResponseWriter, req *http.Request) {}

func newServer(handler http.Handler, port int) (*Server, error) {
	serverSettings := Settings{
		Port:            port,
		RequestTimeout:  time.Second * 3,
		ShutdownTimeout: time.Second * 3,
	}
	api := &testAPI{}
	testCtl := &testController{}

	route := rest.Route{
		Endpoint: rest.Endpoint{
			Path:   "/",
			Method: http.MethodGet,
		},
		Handler: handler,
	}
	testCtl.RegisterRoutes(route)
	api.RegisterControllers(testCtl)
	return New(api, serverSettings)
}

type testAPI struct {
	controllers []rest.Controller
}

func (t *testAPI) Controllers() []rest.Controller {
	return t.controllers
}

func (t *testAPI) RegisterControllers(controllers ...rest.Controller) {
	t.controllers = append(t.controllers, controllers...)
}

type testController struct {
	testRoutes []rest.Route
}

func (t *testController) RegisterRoutes(routes ...rest.Route) {
	t.testRoutes = append(t.testRoutes, routes...)
}

func (t *testController) Routes() []rest.Route {
	return t.testRoutes
}

type testHTTPHandler struct{}

func (t *testHTTPHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Query().Get("fail") == "true" {
		panic("expected")
	}
	rw.WriteHeader(http.StatusOK)
}
