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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/gofrs/uuid"

	"github.com/gorilla/mux"
)

const brokerServerIDPathParam = "broker_server_id"

var brokerNotFoundResponseError = Object{"error": "broker not found"}

func NewBrokerFactory() *BrokerFactory {
	brokerFactory := &BrokerFactory{}
	brokerFactory.brokers = make(map[string]*BrokerServer)
	brokerFactory.initRouter()
	brokerFactory.server = httptest.NewServer(brokerFactory.router)
	return brokerFactory
}

type BrokerFactory struct {
	server  *httptest.Server
	brokers map[string]*BrokerServer
	router  *mux.Router
}

func (bf *BrokerFactory) Close() {
	if bf == nil {
		return
	}
	bf.brokers = make(map[string]*BrokerServer)
	bf.server.Close()
}

func (bf *BrokerFactory) NewBrokerServer() *BrokerServer {
	return bf.NewBrokerServerWithCatalog(NewRandomSBCatalog())
}

func (bf *BrokerFactory) NewBrokerServerWithCatalog(catalog SBCatalog) *BrokerServer {
	brokerServerID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	brokerURL := bf.brokerURLFromBrokerServerID(brokerServerID.String())
	broker := &BrokerServer{
		url:           brokerURL,
		Catalog:       catalog,
		brokerFactory: bf,
	}
	broker.Reset()

	bf.brokers[brokerURL] = broker

	return broker
}

func (bf *BrokerFactory) StopBroker(broker *BrokerServer) {
	if bf == nil {
		return
	}
	delete(bf.brokers, broker.URL())
}

func (bf *BrokerFactory) brokerURLFromBrokerServerID(brokerServerID string) string {
	return fmt.Sprintf("%s/%s", bf.server.URL, brokerServerID)
}

func (bf *BrokerFactory) getBroker(req *http.Request) *BrokerServer {
	brokerServerID := mux.Vars(req)[brokerServerIDPathParam]
	if brokerServerID == "" {
		return nil
	}
	return bf.brokers[bf.brokerURLFromBrokerServerID(brokerServerID)]
}

func (bf *BrokerFactory) initRouter() {
	router := mux.NewRouter()
	router.HandleFunc(fmt.Sprintf("/{%s}/v2/catalog", brokerServerIDPathParam), func(rw http.ResponseWriter, req *http.Request) {
		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}
		broker.CatalogEndpointRequests = append(broker.CatalogEndpointRequests, req)
		broker.CatalogHandler(rw, req)
	}).Methods(http.MethodGet)

	router.HandleFunc(fmt.Sprintf("/{%s}/v2/service_instances/{instance_id}", brokerServerIDPathParam), func(rw http.ResponseWriter, req *http.Request) {
		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}
		broker.ServiceInstanceEndpointRequests = append(broker.ServiceInstanceEndpointRequests, req)
		broker.ServiceInstanceHandler(rw, req)
	}).Methods(http.MethodPut, http.MethodDelete, http.MethodGet, http.MethodPatch)

	router.HandleFunc(fmt.Sprintf("/{%s}/v2/service_instances/{instance_id}/service_bindings/{binding_id}", brokerServerIDPathParam), func(rw http.ResponseWriter, req *http.Request) {
		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}
		broker.BindingEndpointRequests = append(broker.BindingEndpointRequests, req)
		broker.BindingHandler(rw, req)
	}).Methods(http.MethodPut, http.MethodDelete, http.MethodGet)

	router.HandleFunc(fmt.Sprintf("/{%s}/v2/service_instances/{instance_id}/last_operation", brokerServerIDPathParam), func(rw http.ResponseWriter, req *http.Request) {
		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}
		broker.ServiceInstanceLastOpEndpointRequests = append(broker.ServiceInstanceLastOpEndpointRequests, req)
		broker.ServiceInstanceLastOpHandler(rw, req)
	}).Methods(http.MethodGet)

	router.HandleFunc(fmt.Sprintf("/{%s}/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", brokerServerIDPathParam), func(rw http.ResponseWriter, req *http.Request) {
		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}
		broker.BindingLastOpEndpointRequests = append(broker.BindingLastOpEndpointRequests, req)
		broker.BindingLastOpHandler(rw, req)
	}).Methods(http.MethodGet)

	router.HandleFunc(fmt.Sprintf("/{%s}/v2/service_instances/{instance_id}/service_bindings/{binding_id}/adapt_credentials", brokerServerIDPathParam), func(rw http.ResponseWriter, req *http.Request) {
		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}
		broker.BindingAdaptCredentialsEndpointRequests = append(broker.BindingAdaptCredentialsEndpointRequests, req)
		broker.BindingAdaptCredentialsHandler(rw, req)
	}).Methods(http.MethodPost)

	router.Use(bf.authenticationMiddleware)
	router.Use(bf.saveRequestMiddleware)

	bf.router = router
}

func (bf *BrokerFactory) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		auth := req.Header.Get("Authorization")
		if auth == "" {
			SetResponse(rw, http.StatusUnauthorized, Object{"error": "Missing authorization header"})
			return
		}
		const basicHeaderPrefixLength = len("Basic ")
		decoded, err := base64.StdEncoding.DecodeString(auth[basicHeaderPrefixLength:])
		if err != nil {
			SetResponse(rw, http.StatusUnauthorized, Object{
				"error": err.Error(),
			})
			return
		}

		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}

		if string(decoded) != fmt.Sprintf("%s:%s", broker.Username, broker.Password) {
			SetResponse(rw, http.StatusUnauthorized, Object{"error": "Credentials mismatch"})
			return
		}
		next.ServeHTTP(rw, req)
	})
}

func (bf *BrokerFactory) saveRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer func() {
			err := req.Body.Close()
			if err != nil {
				panic(err)
			}
		}()
		broker := bf.getBroker(req)
		if broker == nil {
			SetResponse(rw, http.StatusNotFound, brokerNotFoundResponseError)
			return
		}
		broker.LastRequest = req
		bodyBytes, err := ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		if err != nil {
			SetResponse(rw, http.StatusInternalServerError, Object{
				"description": "Could not read body",
				"error":       err.Error(),
			})
			return
		}
		broker.LastRequestBody = bodyBytes
		next.ServeHTTP(rw, req)
	})
}
