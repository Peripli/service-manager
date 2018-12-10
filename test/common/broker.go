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

const Catalog = `
{
  "services": [{
    "name": "fake-service",
    "id": "acb56d7c-XXXX-XXXX-XXXX-feb140a59a66",
    "description": "A fake service.",
    "tags": ["no-sql", "relational"],
    "requires": ["route_forwarding"],
    "bindable": true,
    "instances_retrievable": true,
    "bindings_retrievable": true,
    "metadata": {
      "provider": {
        "name": "The name"
      },
      "listing": {
        "imageUrl": "http://example.com/cat.gif",
        "blurb": "Add a blurb here",
        "longDescription": "A long time ago, in a galaxy far far away..."
      },
      "displayName": "The Fake Service Broker"
    },
    "plan_updateable": true,
    "plans": [{
      "name": "fake-plan-1",
      "id": "d3031751-XXXX-XXXX-XXXX-a42377d3320e",
      "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections.",
      "free": false,
      "metadata": {
        "max_storage_tb": 5,
        "costs":[
            {
               "amount":{
                  "usd":99.0
               },
               "unit":"MONTHLY"
            },
            {
               "amount":{
                  "usd":0.99
               },
               "unit":"1GB of messages over 20GB"
            }
         ],
        "bullets": [
          "Shared fake server",
          "5 TB storage",
          "40 concurrent connections"
        ]
      },
      "schemas": {
        "service_instance": {
          "create": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "type": "object",
              "properties": {
                "billing-account": {
                  "description": "Billing account number used to charge use of shared fake server.",
                  "type": "string"
                }
              }
            }
          },
          "update": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "type": "object",
              "properties": {
                "billing-account": {
                  "description": "Billing account number used to charge use of shared fake server.",
                  "type": "string"
                }
              }
            }
          }
        },
        "service_binding": {
          "create": {
            "parameters": {
              "$schema": "http://json-schema.org/draft-04/schema#",
              "type": "object",
              "properties": {
                "billing-account": {
                  "description": "Billing account number used to charge use of shared fake server.",
                  "type": "string"
                }
              }
            }
          }
        }
      }
    }, 
	{
      "name": "fake-plan-2",
      "id": "0f4008b5-XXXX-XXXX-XXXX-dace631cd648",
      "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections. 100 async.",
      "free": false,
      "metadata": {
        "max_storage_tb": 5,
        "costs":[
            {
               "amount":{
                  "usd":199.0
               },
               "unit":"MONTHLY"
            },
            {
               "amount":{
                  "usd":0.99
               },
               "unit":"1GB of messages over 20GB"
            }
         ],
        "bullets": [
          "40 concurrent connections"
        ]
      }
    }]
  }]
}
`

const AnotherService = `
{
    "name": "another-fake-service",
    "id": "another7c-XXXX-XXXX-XXXX-feb140a59a66",
    "description": "another description",
    "requires": ["another-route_forwarding"],
    "tags": ["another-no-sql", "another-relational"],
    "bindable": true,	
    "instances_retrievable": true,	
    "bindings_retrievable": true,	
    "metadata": {	
      "provider": {	
        "name": "another name"	
      },	
      "listing": {	
        "imageUrl": "http://example.com/cat.gif",	
        "blurb": "another blurb here",	
        "longDescription": "A long time ago, in a another galaxy far far away..."	
      },	
      "displayName": "another Fake Service Broker"	
    },	
    "plan_updateable": true,	
    "plans": []
}
`

const AnotherPlan = `
	{
      "name": "another-fake-plan",
      "id": "123008b5-XXXX-XXXX-XXXX-dace631cd648",
      "description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections. 100 async.",
      "free": false,
      "metadata": {
        "max_storage_tb": 5,
        "costs":[
            {
               "amount":{
                  "usd":199.0
               },
               "unit":"MONTHLY"
            },
            {
               "amount":{
                  "usd":0.99
               },
               "unit":"1GB of messages over 20GB"
            }
         ],
        "bullets": [
          "40 concurrent connections"
        ]
      }
    }
`

type BrokerServer struct {
	*httptest.Server

	CatalogHandler                 http.HandlerFunc // /v2/catalog
	ServiceInstanceHandler         http.HandlerFunc // /v2/service_instances/{instance_id}
	ServiceInstanceLastOpHandler   http.HandlerFunc // /v2/service_instances/{instance_id}/last_operation
	BindingHandler                 http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}
	BindingLastOpHandler           http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation
	BindingAdaptCredentialsHandler http.HandlerFunc // /v2/service_instances/{instance_id}/service_bindings/{binding_id}/adapt_credentials

	Username, Password string
	Catalog            interface{}
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

func DefaultCatalog() map[string]interface{} {
	return JSONToMap(Catalog)
}

func NewBrokerServer() *BrokerServer {
	brokerServer := &BrokerServer{}
	brokerServer.initRouter()
	brokerServer.Reset()

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
	b.Catalog = DefaultCatalog()
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
	SetResponse(rw, http.StatusOK, b.Catalog)
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
