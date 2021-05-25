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
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/Peripli/service-manager/test/tls_settings"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/gorilla/mux"
)

type BrokerServer struct {
	*httptest.Server

	CatalogHandler               http.HandlerFunc // /v2/catalog
	ServiceInstanceHandler       http.HandlerFunc // Provision/v2/service_instances/{instance_id}
	ServiceInstanceLastOpHandler http.HandlerFunc // /v2/service_instances/{instance_id}/last_operation
	ServiceInstanceOperations    []string

	BindingHandler                 http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}
	BindingAdaptCredentialsHandler http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}/adapt_credentials
	BindingLastOpHandler           http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation
	ServiceBindingOperations       []string

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

	mutex *sync.RWMutex

	shouldRecordRequests bool
}

func (b *BrokerServer) URL() string {
	return b.Server.URL
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

func NewBrokerServerTLS() *BrokerServer {
	return NewBrokerServerWithTLSAndCatalog(NewRandomSBCatalog())
}

func NewBrokerServerWithCatalog(catalog SBCatalog) *BrokerServer {
	brokerServer := &BrokerServer{}
	brokerServer.mutex = &sync.RWMutex{}
	brokerServer.shouldRecordRequests = true
	brokerServer.initRouter()
	brokerServer.Reset()
	brokerServer.Catalog = catalog
	brokerServer.Server = httptest.NewServer(brokerServer.router)
	return brokerServer
}

func NewBrokerServerWithTLSAndCatalog(catalog SBCatalog) *BrokerServer {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(tls_settings.ClientCertificate))
	brokerServer := &BrokerServer{}
	brokerServer.mutex = &sync.RWMutex{}
	brokerServer.shouldRecordRequests = true
	brokerServer.initRouter()
	brokerServer.Reset()
	brokerServer.Catalog = catalog
	uServer := httptest.NewUnstartedServer(brokerServer.router)
	uServer.TLS = &tls.Config{}
	uServer.TLS.ClientCAs = caCertPool
	uServer.TLS.ClientAuth = tls.RequireAndVerifyClientCert
	brokerServer.Server = uServer
	brokerServer.StartTLS()
	return brokerServer
}

func (b *BrokerServer) ShouldRecordRequests(shouldRecordRequests bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.shouldRecordRequests = shouldRecordRequests
}

func (b *BrokerServer) Reset() {
	b.ResetProperties()
	b.ResetHandlers()
	b.ResetCallHistory()
}

func (b *BrokerServer) ResetProperties() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.Username = "admin"
	b.Password = "admin"
	c := NewRandomSBCatalog()
	b.Catalog = c
	b.LastRequestBody = []byte{}
	b.LastRequest = nil
}

func (b *BrokerServer) ResetHandlers() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.CatalogHandler = b.defaultCatalogHandler
	b.ServiceInstanceHandler = b.defaultServiceInstanceHandler
	b.ServiceInstanceLastOpHandler = b.defaultServiceInstanceLastOpHandler
	b.BindingHandler = b.defaultBindingHandler
	b.BindingLastOpHandler = b.defaultBindingLastOpHandler
	b.BindingAdaptCredentialsHandler = b.defaultBindingAdaptCredentialsHandler
}

func (b *BrokerServer) ResetCallHistory() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.CatalogEndpointRequests = make([]*http.Request, 0)
	b.ServiceInstanceEndpointRequests = make([]*http.Request, 0)
	b.ServiceInstanceLastOpEndpointRequests = make([]*http.Request, 0)
	b.BindingEndpointRequests = make([]*http.Request, 0)
	b.BindingLastOpEndpointRequests = make([]*http.Request, 0)
}

func (b *BrokerServer) initRouter() {
	router := mux.NewRouter()
	router.HandleFunc("/v2/catalog", func(rw http.ResponseWriter, req *http.Request) {
		b.mutex.RLock()
		b.CatalogHandler(rw, req)
		b.mutex.RUnlock()

		if b.shouldRecordRequests {
			b.mutex.Lock()
			b.CatalogEndpointRequests = append(b.CatalogEndpointRequests, req)
			b.mutex.Unlock()
		}
	}).Methods(http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		b.mutex.RLock()
		b.ServiceInstanceHandler(rw, req)
		b.mutex.RUnlock()

		if b.shouldRecordRequests {
			b.mutex.Lock()
			b.ServiceInstanceEndpointRequests = append(b.ServiceInstanceEndpointRequests, req)
			b.mutex.Unlock()
		}
	}).Methods(http.MethodPut, http.MethodDelete, http.MethodGet, http.MethodPatch)

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		b.mutex.RLock()
		b.BindingHandler(rw, req)
		b.mutex.RUnlock()

		if b.shouldRecordRequests {
			b.mutex.Lock()
			b.BindingEndpointRequests = append(b.BindingEndpointRequests, req)
			b.mutex.Unlock()
		}
	}).Methods(http.MethodPut, http.MethodDelete, http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", func(rw http.ResponseWriter, req *http.Request) {
		b.mutex.RLock()
		b.ServiceInstanceLastOpHandler(rw, req)
		b.mutex.RUnlock()

		if b.shouldRecordRequests {
			b.mutex.Lock()
			b.ServiceInstanceLastOpEndpointRequests = append(b.ServiceInstanceLastOpEndpointRequests, req)
			b.mutex.Unlock()
		}
	}).Methods(http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", func(rw http.ResponseWriter, req *http.Request) {
		b.mutex.RLock()
		b.BindingLastOpHandler(rw, req)
		b.mutex.RUnlock()

		if b.shouldRecordRequests {
			b.mutex.Lock()
			b.BindingLastOpEndpointRequests = append(b.BindingLastOpEndpointRequests, req)
			b.mutex.Unlock()
		}
	}).Methods(http.MethodGet)

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/adapt_credentials", func(rw http.ResponseWriter, req *http.Request) {
		b.mutex.RLock()
		b.BindingAdaptCredentialsHandler(rw, req)
		b.mutex.RUnlock()

		if b.shouldRecordRequests {
			b.mutex.Lock()
			b.BindingAdaptCredentialsEndpointRequests = append(b.BindingAdaptCredentialsEndpointRequests, req)
			b.mutex.Unlock()
		}
	}).Methods(http.MethodPost)

	router.Use(b.authenticationMiddleware)
	router.Use(b.saveRequestMiddleware)
	b.router = router
}

func (b *BrokerServer) ServiceInstanceHandlerFunc(method, op string, handler func(req *http.Request) (int, map[string]interface{})) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	currentHandler := b.ServiceInstanceHandler
	b.ServiceInstanceHandler = func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == method {
			status, body := handler(req)
			body["operation"] = op
			SetResponse(rw, status, body)
			return
		}
		currentHandler(rw, req)
	}
}

func (b *BrokerServer) ServiceInstanceLastOpHandlerFunc(op string, handler func(req *http.Request) (int, map[string]interface{})) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	currentHandler := b.ServiceInstanceLastOpHandler
	b.ServiceInstanceLastOpHandler = func(rw http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			panic(err)
		}
		opFromBody := req.Form["operation"][0]
		if opFromBody == op {
			status, body := handler(req)
			SetResponse(rw, status, body)
			return
		}
		currentHandler(rw, req)
	}
}

func (b *BrokerServer) BindingHandlerFunc(method, op string, handler func(req *http.Request) (int, map[string]interface{})) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	currentHandler := b.BindingHandler
	b.BindingHandler = func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == method {
			status, body := handler(req)
			body["operation"] = op
			SetResponse(rw, status, body)
			return
		}
		currentHandler(rw, req)
	}
}

func (b *BrokerServer) BindingLastOpHandlerFunc(op string, handler func(req *http.Request) (int, map[string]interface{})) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	currentHandler := b.BindingLastOpHandler
	b.BindingLastOpHandler = func(rw http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			panic(err)
		}
		opFromBody := req.Form["operation"][0]
		if opFromBody == op {
			status, body := handler(req)
			SetResponse(rw, status, body)
			return
		}
		currentHandler(rw, req)
	}
}

func (b *BrokerServer) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if b.Server.TLS == nil {
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
		}
		next.ServeHTTP(w, req)
	})
}

func (b *BrokerServer) saveRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !b.shouldRecordRequests {
			next.ServeHTTP(w, req)
			return
		}

		defer func() {
			err := req.Body.Close()
			if err != nil {
				panic(err)
			}
		}()
		b.mutex.Lock()
		b.LastRequest = req
		b.mutex.Unlock()
		bodyBytes, err := ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		if err != nil {
			SetResponse(w, http.StatusInternalServerError, Object{
				"description": "Could not read body",
				"error":       err.Error(),
			})
			return
		}
		b.mutex.Lock()
		b.LastRequestBody = bodyBytes
		b.mutex.Unlock()
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
				"user":     "user",
				"password": "password",
			},
		})
	} else if req.Method == http.MethodGet {
		SetResponse(rw, http.StatusOK, Object{
			"credentials": Object{
				"user":     "user",
				"password": "password",
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
			"user":     "user",
			"password": "password",
		},
	})
}

func SetResponse(rw http.ResponseWriter, status int, message map[string]interface{}) {
	err := util.WriteJSON(rw, status, message)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
	}
}

func DelayingHandler(done chan interface{}) func(req *http.Request) (int, map[string]interface{}) {
	return func(req *http.Request) (int, map[string]interface{}) {
		timeout := time.After(10 * time.Second)
		select {
		case <-done:
		case <-timeout:
		}

		return http.StatusTeapot, Object{}
	}
}

func ParameterizedHandler(statusCode int, responseBody map[string]interface{}) func(_ *http.Request) (int, map[string]interface{}) {
	return func(_ *http.Request) (int, map[string]interface{}) {
		return statusCode, responseBody
	}
}

func MultiplePollsRequiredHandler(state, finalState string) func(_ *http.Request) (int, map[string]interface{}) {
	polls := 0
	return func(_ *http.Request) (int, map[string]interface{}) {
		if polls == 0 {
			polls++
			return http.StatusOK, Object{"state": state}
		} else {
			return http.StatusOK, Object{"state": finalState}
		}
	}
}

func MultipleErrorsBeforeSuccessHandler(initialStatusCode, finalStatusCode int, initialBody, finalBody Object) func(_ *http.Request) (int, map[string]interface{}) {
	repeats := 0
	return func(_ *http.Request) (int, map[string]interface{}) {
		if repeats == 0 {
			repeats++
			return initialStatusCode, initialBody
		} else {
			return finalStatusCode, finalBody
		}
	}
}
