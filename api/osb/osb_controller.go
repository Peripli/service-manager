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

package osb

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"regexp"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

var osbPathPattern = regexp.MustCompile("^" + web.OSBURL + "/[^/]+(/.*)$")

// BrokerFetcher is implemented by OSB handler providers
type BrokerFetcher interface {
	FetchBroker(ctx context.Context, brokerID string) (*types.Broker, error)
}

// controller implements api.Controller by providing OSB API logic
type controller struct {
	fetcher BrokerFetcher
}

var _ web.Controller = &controller{}

// NewController returns new OSB controller
func NewController(fetcher BrokerFetcher, _ http.RoundTripper) web.Controller {
	return &controller{
		fetcher: fetcher,
	}
}

func (c *controller) handler(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logger := log.C(ctx)
	logger.Debug("Executing OSB operation: ", r.URL.Path)

	brokerID, ok := r.PathParams[BrokerIDPathParam]
	if !ok {
		logger.Debugf("error creating OSB client: brokerID path parameter not found")
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "invalid broker id path parameter",
			StatusCode:  http.StatusBadRequest,
		}
	}
	logger.Debugf("Obtained path parameter [brokerID = %s] from path params", brokerID)

	broker, err := c.fetcher.FetchBroker(ctx, brokerID)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Fetched broker %s with id %s accessible at %s", broker.ID, broker.Name, broker.BrokerURL)

	targetBrokerURL, _ := url.Parse(broker.BrokerURL)

	m := osbPathPattern.FindStringSubmatch(r.URL.Path)
	if m == nil || len(m) < 2 {
		return nil, fmt.Errorf("could not get OSB path from URL %s", r.URL)
	}

	modifiedRequest := r.Request.WithContext(ctx)
	modifiedRequest.SetBasicAuth(broker.Credentials.Basic.Username, broker.Credentials.Basic.Password)
	modifiedRequest.Body = ioutil.NopCloser(bytes.NewReader(r.Body))
	modifiedRequest.ContentLength = int64(len(r.Body))
	modifiedRequest.URL.Path = m[1]

	// This is needed because the request is shallow copy of the request to the Service Manager
	// This sets the host header to point to the service broker that the request will be proxied to
	modifiedRequest.Host = targetBrokerURL.Host

	proxy := httputil.NewSingleHostReverseProxy(targetBrokerURL)

	director := proxy.Director
	proxy.Director = func(request *http.Request) {
		director(request)
		logger.Debugf("Forwarded OSB request to service broker %s at %s", broker.Name, request.URL)
	}
	proxy.ModifyResponse = func(response *http.Response) error {
		logger.Debugf("Service broker %s replied with status %d", broker.Name, response.StatusCode)
		return nil
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
		logger.WithError(e).Errorf("Error while forwarding request to service broker %s", broker.Name)
		util.WriteError(&util.HTTPError{
			ErrorType:   "ServiceBrokerErr",
			Description: fmt.Sprintf("could not reach service broker %s at %s", broker.Name, request.URL),
			StatusCode:  http.StatusBadGateway,
		}, writer)
	}

	recorder := httptest.NewRecorder()

	proxy.ServeHTTP(recorder, modifiedRequest)

	respBody, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		return nil, err
	}

	resp := &web.Response{
		StatusCode: recorder.Code,
		Header:     recorder.Header(),
		Body:       respBody,
	}
	return resp, nil
}
