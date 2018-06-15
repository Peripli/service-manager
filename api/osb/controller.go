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

// Package osb contains logic for building the Service Manager OSB API
package osb

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"regexp"

	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/types"
	"github.com/sirupsen/logrus"
)

const (

	// v1 is a prefix the first version of the OSB API
	v1 = "/v1"

	// root is a prefix for the OSB API
	root = "/osb"

	//BrokerIDPathParam is a service broker ID path parameter
	BrokerIDPathParam = "brokerID"

	// baseURL is the OSB API controller path
	baseURL = v1 + root + "/{" + BrokerIDPathParam + "}"

	catalogURL         = baseURL + "/v2/catalog"
	serviceInstanceURL = baseURL + "/v2/service_instances/{instance_id}"
	serviceBindingURL  = baseURL + "/v2/service_instances/{instance_id}/service_bindings/{binding_id}"
)

var osbPattern = regexp.MustCompile("^" + v1 + root + "/[^/]+(/.*)$")

// Controller implements rest.Controller by providing OSB API logic
type Controller struct {
	BrokerStorage storage.Broker
}

var _ rest.Controller = &Controller{}

// Routes implements rest.Controller.Routes by providing the routes for the OSB API
func (c *Controller) Routes() []rest.Route {
	return []rest.Route{
		{rest.Endpoint{"GET", catalogURL}, c.handler},
		{rest.Endpoint{"PUT", serviceInstanceURL}, c.handler},
		{rest.Endpoint{"DELETE", serviceInstanceURL}, c.handler},
		{rest.Endpoint{"PUT", serviceBindingURL}, c.handler},
		{rest.Endpoint{"DELETE", serviceBindingURL}, c.handler},
	}
}

func (c *Controller) handler(request *filter.Request) (*filter.Response, error) {
	broker, err := c.fetchBroker(request)
	if err != nil {
		return nil, err
	}
	target, _ := url.Parse(broker.BrokerURL)

	reverseProxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Header.Set("Host", target.Host)
		},
	}

	modifiedRequest := request.Request.WithContext(request.Context())
	modifiedRequest.Header.Add("Authorization", basicAuth(broker.Credentials.Basic))
	modifiedRequest.URL.Scheme = target.Scheme
	modifiedRequest.URL.Host = target.Host

	m := osbPattern.FindStringSubmatch(request.URL.Path)
	if m == nil || len(m) < 2 {
		return nil, fmt.Errorf("Could not get OSB path from URL %s", request.URL.Path)
	}
	modifiedRequest.URL.Path = m[1]

	logrus.Debugf("Forwarding OSB request to %s", modifiedRequest.URL)
	recorder := httptest.NewRecorder()
	modifiedRequest.Body = ioutil.NopCloser(bytes.NewReader(request.Body))
	reverseProxy.ServeHTTP(recorder, modifiedRequest)

	body, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		return nil, err
	}

	headers := recorder.HeaderMap
	resp := &filter.Response{
		StatusCode: recorder.Code,
		Body:       body,
		Header:     headers,
	}
	logrus.Debugf("Service broker replied with status %d", resp.StatusCode)
	return resp, nil
}

func basicAuth(credentials *types.Basic) string {
	return "Basic " + base64.StdEncoding.EncodeToString(
		[]byte(credentials.Username+":"+credentials.Password))
}

func (c *Controller) fetchBroker(request *filter.Request) (*types.Broker, error) {
	brokerID, ok := request.PathParams[BrokerIDPathParam]
	if !ok {
		logrus.Debugf("error creating OSB client: brokerID path parameter not found")
		return nil, types.NewErrorResponse(errors.New("Invalid broker id path parameter"), http.StatusBadRequest, "BadRequest")
	}
	logrus.Debugf("Obtained path parameter [brokerID = %s] from path params", brokerID)

	serviceBroker, err := c.BrokerStorage.Get(brokerID)
	if err == storage.ErrNotFound {
		logrus.Debugf("service broker with id %s not found", brokerID)
		return nil, types.NewErrorResponse(fmt.Errorf("Could not find broker with id: %s", brokerID), http.StatusNotFound, "NotFound")
	} else if err != nil {
		logrus.Errorf("error obtaining serviceBroker with id %s from storage: %s", brokerID, err)
		return nil, fmt.Errorf("Internal Server Error")
	}

	return serviceBroker, nil
}
