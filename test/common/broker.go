package common

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

type Broker struct {
	StatusCode     int
	ResponseBody   []byte
	Request        *http.Request
	RequestBody    *httpexpect.Value
	RawRequestBody []byte
	OSBURL         string
	Server         *httptest.Server
	ID             string
}

const serviceCatalog = `{
	"services": [{
		"id": "1234",
		"name": "service1",
		"description": "sample-test",
		"bindable": true,
		"plans": [{
			"id": "plan-id",
			"name": "plan-name",
			"description": "plan-desc"
		}]
	}]
}`

func (b *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.Request = req
	responseBody := b.ResponseBody
	switch req.Method {
	case http.MethodPatch, http.MethodPost, http.MethodPut:
		var err error
		b.RawRequestBody, err = ioutil.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}
		var reqData interface{}
		err = json.Unmarshal(b.RawRequestBody, &reqData)
		if err != nil {
			panic(err)
		}
		b.RequestBody = httpexpect.NewValue(GinkgoT(), reqData)

	case http.MethodGet:
		if responseBody == nil && req.URL.Path == "/v2/catalog" {
			responseBody = []byte(serviceCatalog)
		}
	}

	code := b.StatusCode
	if code == 0 {
		code = http.StatusOK
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)

	rw.Write(responseBody)
}

func (b *Broker) Called() bool {
	return b.Request != nil
}
