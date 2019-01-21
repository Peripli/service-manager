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

package common

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/gorilla/mux"
)

type BrokerServer struct {
	*httptest.Server

	CatalogHandler                 http.HandlerFunc // /v2/catalog
	ServiceInstanceHandler         http.HandlerFunc // /v2/service_instances/{instance_id}
	ServiceInstanceLastOpHandler   http.HandlerFunc // /v2/service_instances/{instance_id}/last_operation
	BindingHandler                 http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}
	BindingLastOpHandler           http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation
	BindingAdaptCredentialsHandler http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}/adapt_credentials

	Username, Password string
	Catalog            SBCatalog
	LastRequestBody    []byte
	LastRequest        *http.Request

	CatalogEndpointRequests                 []*http.Request
	ServiceInstanceEndpointRequests         []*http.Request
	ServiceInstanceLastOpEndpointRequests   []*http.Request
	BindingEndpointRequests                 []*http.Request
	BindingLastOpEndpointRequests           []*http.Request
	BindingAdaptCredentialsEndpointRequests []*http.Request

	router *mux.Router
}

func JSONToMap(j string) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(j), &jsonMap); err != nil {
		panic(err)
	}
	return jsonMap
}

func NewBrokerServer() *BrokerServer {
	return NewBrokerServerWithCatalog(NewRandomSBCatalog())
}
func NewBrokerServerWithCatalog(catalog SBCatalog) *BrokerServer {
	brokerServer := &BrokerServer{}
	brokerServer.initRouter()
	brokerServer.Reset()
	brokerServer.Catalog = catalog
	brokerServer.Server = httptest.NewServer(brokerServer.router)
	return brokerServer
}

func (b *BrokerServer) Reset() {
	b.ResetProperties()
	b.ResetHandlers()
	b.ResetCallHistory()
}

func (b *BrokerServer) ResetProperties() {
	b.Username = "buser"
	b.Password = "bpassword"
	c := NewRandomSBCatalog()
	b.Catalog = c
	b.LastRequestBody = []byte{}
	b.LastRequest = nil
}

func (b *BrokerServer) ResetHandlers() {
	b.CatalogHandler = b.defaultCatalogHandler
	b.ServiceInstanceHandler = b.defaultServiceInstanceHandler
	b.ServiceInstanceLastOpHandler = b.defaultServiceInstanceLastOpHandler
	b.BindingHandler = b.defaultBindingHandler
	b.BindingLastOpHandler = b.defaultBindingLastOpHandler
	b.BindingAdaptCredentialsHandler = b.defaultBindingAdaptCredentialsHandler
}

func (b *BrokerServer) ResetCallHistory() {
	b.CatalogEndpointRequests = make([]*http.Request, 0)
	b.ServiceInstanceEndpointRequests = make([]*http.Request, 0)
	b.ServiceInstanceLastOpEndpointRequests = make([]*http.Request, 0)
	b.BindingEndpointRequests = make([]*http.Request, 0)
	b.BindingLastOpEndpointRequests = make([]*http.Request, 0)
}

func (b *BrokerServer) initRouter() {
	router := mux.NewRouter()
	router.HandleFunc("/v2/catalog", func(rw http.ResponseWriter, req *http.Request) {
		b.CatalogEndpointRequests = append(b.CatalogEndpointRequests, req)
		b.CatalogHandler(rw, req)
	}).Methods(http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		b.ServiceInstanceEndpointRequests = append(b.ServiceInstanceEndpointRequests, req)
		b.ServiceInstanceHandler(rw, req)
	}).Methods(http.MethodPut, http.MethodDelete, http.MethodGet, http.MethodPatch)

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		b.BindingEndpointRequests = append(b.BindingEndpointRequests, req)
		b.BindingHandler(rw, req)
	}).Methods(http.MethodPut, http.MethodDelete, http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", func(rw http.ResponseWriter, req *http.Request) {
		b.ServiceInstanceLastOpEndpointRequests = append(b.ServiceInstanceLastOpEndpointRequests, req)
		b.ServiceInstanceLastOpHandler(rw, req)
	}).Methods(http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", func(rw http.ResponseWriter, req *http.Request) {
		b.BindingLastOpEndpointRequests = append(b.BindingLastOpEndpointRequests, req)
		b.BindingLastOpHandler(rw, req)
	}).Methods(http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/adapt_credentials", func(rw http.ResponseWriter, req *http.Request) {
		b.BindingAdaptCredentialsEndpointRequests = append(b.BindingAdaptCredentialsEndpointRequests, req)
		b.BindingAdaptCredentialsHandler(rw, req)
	}).Methods(http.MethodPost)

	router.Use(b.authenticationMiddleware)
	router.Use(b.saveRequestMiddleware)

	b.router = router
}

func (b *BrokerServer) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		auth := req.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing authorization header"))
			return
		}
		const basicHeaderPrefixLength = len("Basic ")
		decoded, err := base64.StdEncoding.DecodeString(auth[basicHeaderPrefixLength:])
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}
		if string(decoded) != fmt.Sprintf("%s:%s", b.Username, b.Password) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Credentials mismatch"))
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (b *BrokerServer) saveRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b.LastRequest = req
		bodyBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			SetResponse(w, http.StatusInternalServerError, Object{
				"description": "Could not read body",
				"error":       err.Error(),
			})
			return
		}
		b.LastRequestBody = bodyBytes
		next.ServeHTTP(w, req)
	})
}

func (b *BrokerServer) defaultCatalogHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, JSONToMap(string(b.Catalog)))
}

func (b *BrokerServer) defaultServiceInstanceHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPut {
		SetResponse(rw, http.StatusCreated, Object{})
	} else {
		SetResponse(rw, http.StatusOK, Object{})
	}
}

func (b *BrokerServer) defaultServiceInstanceLastOpHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, Object{
		"state": "succeeded",
	})
}

func (b *BrokerServer) defaultBindingHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPut {
		SetResponse(rw, http.StatusCreated, Object{
			"credentials": Object{
				"instance_id": mux.Vars(req)["instance_id"],
				"binding_id":  mux.Vars(req)["binding_id"],
			},
		})
	} else {
		SetResponse(rw, http.StatusOK, Object{})
	}
}

func (b *BrokerServer) defaultBindingLastOpHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, Object{
		"state": "succeeded",
	})
}

func (b *BrokerServer) defaultBindingAdaptCredentialsHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, Object{
		"credentials": Object{
			"instance_id": mux.Vars(req)["instance_id"],
			"binding_id":  mux.Vars(req)["binding_id"],
		},
	})
}

func SetResponse(rw http.ResponseWriter, status int, message interface{}) {
	err := util.WriteJSON(rw, status, message)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
	}
}
