package common

import (
	"encoding/json"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/gorilla/mux"
)

type BrokerServer struct {
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
	url                string
	brokerFactory      *BrokerFactory

	CatalogEndpointRequests                 []*http.Request
	ServiceInstanceEndpointRequests         []*http.Request
	ServiceInstanceLastOpEndpointRequests   []*http.Request
	BindingEndpointRequests                 []*http.Request
	BindingLastOpEndpointRequests           []*http.Request
	BindingAdaptCredentialsEndpointRequests []*http.Request
}

func (tb *BrokerServer) URL() string {
	return tb.url
}

func (tb *BrokerServer) Reset() {
	tb.ResetProperties()
	tb.ResetHandlers()
	tb.ResetCallHistory()
}

func (tb *BrokerServer) Close() {
	if tb == nil {
		return
	}
	tb.brokerFactory.StopBroker(tb)
}

func (tb *BrokerServer) ResetProperties() {
	tb.Username = "buser"
	tb.Password = "bpassword"
	tb.LastRequestBody = []byte{}
	tb.LastRequest = nil
}

func (tb *BrokerServer) ResetHandlers() {
	tb.CatalogHandler = tb.defaultCatalogHandler
	tb.ServiceInstanceHandler = tb.defaultServiceInstanceHandler
	tb.ServiceInstanceLastOpHandler = tb.defaultServiceInstanceLastOpHandler
	tb.BindingHandler = tb.defaultBindingHandler
	tb.BindingLastOpHandler = tb.defaultBindingLastOpHandler
	tb.BindingAdaptCredentialsHandler = tb.defaultBindingAdaptCredentialsHandler
}

func (tb *BrokerServer) ResetCallHistory() {
	tb.CatalogEndpointRequests = make([]*http.Request, 0)
	tb.ServiceInstanceEndpointRequests = make([]*http.Request, 0)
	tb.ServiceInstanceLastOpEndpointRequests = make([]*http.Request, 0)
	tb.BindingEndpointRequests = make([]*http.Request, 0)
	tb.BindingLastOpEndpointRequests = make([]*http.Request, 0)
}

func (tb *BrokerServer) defaultCatalogHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, JSONToMap(string(tb.Catalog)))
}

func (tb *BrokerServer) defaultServiceInstanceHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPut {
		SetResponse(rw, http.StatusCreated, Object{})
	} else {
		SetResponse(rw, http.StatusOK, Object{})
	}
}

func (tb *BrokerServer) defaultServiceInstanceLastOpHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, Object{
		"state": "succeeded",
	})
}

func (tb *BrokerServer) defaultBindingHandler(rw http.ResponseWriter, req *http.Request) {
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

func (tb *BrokerServer) defaultBindingLastOpHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, Object{
		"state": "succeeded",
	})
}

func (tb *BrokerServer) defaultBindingAdaptCredentialsHandler(rw http.ResponseWriter, req *http.Request) {
	SetResponse(rw, http.StatusOK, Object{
		"credentials": Object{
			"instance_id": mux.Vars(req)["instance_id"],
			"binding_id":  mux.Vars(req)["binding_id"],
		},
	})
}

func JSONToMap(j string) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(j), &jsonMap); err != nil {
		panic(err)
	}
	return jsonMap
}

func SetResponse(rw http.ResponseWriter, status int, message interface{}) {
	err := util.WriteJSON(rw, status, message)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, err := rw.Write([]byte(err.Error()))
		if err != nil {
			panic(err)
		}
	}
}
