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

const name = "api/controller/osb"

var osbPathPattern = regexp.MustCompile("^" + web.OSBURL + "/[^/]+(/.*)$")

// BrokerRoundTripper is implemented by OSB handler providers
type BrokerRoundTripper interface {
	http.RoundTripper

	Broker(ctx context.Context, brokerID string) (*types.Broker, error)
}

// Controller implements api.Controller by providing OSB API logic
type controller struct {
	fetcher BrokerRoundTripper
	proxy   *httputil.ReverseProxy
}

var _ web.Controller = &controller{}

// NewController returns new OSB controller
func NewController(fetcher BrokerRoundTripper) web.Controller {
	return &controller{
		proxy: &httputil.ReverseProxy{
			Transport: fetcher,
			Director:  func(req *http.Request) {},
		},
		fetcher: fetcher,
	}
}

func (c *controller) handler(r *web.Request) (*web.Response, error) {
	log.R(r, name).Debug("Executing OSB operation: ", r.URL.Path)

	brokerID, ok := r.PathParams[BrokerIDPathParam]
	if !ok {
		log.R(r, name).Debugf("error creating OSB client: brokerID path parameter not found")
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "invalid broker id path parameter",
			StatusCode:  http.StatusBadRequest,
		}
	}
	log.R(r, name).Debugf("Obtained path parameter [brokerID = %s] from path params", brokerID)

	broker, err := c.fetcher.Broker(r.Context(), brokerID)
	if err != nil {
		return nil, err
	}

	targetBrokerURL, _ := url.Parse(broker.BrokerURL)

	m := osbPathPattern.FindStringSubmatch(r.URL.Path)
	if m == nil || len(m) < 2 {
		return nil, fmt.Errorf("could not get OSB path from URL %s", r.URL.Path)
	}

	modifiedRequest := r.Request.WithContext(r.Context())
	modifiedRequest.SetBasicAuth(broker.Credentials.Basic.Username, broker.Credentials.Basic.Password)
	modifiedRequest.Body = ioutil.NopCloser(bytes.NewReader(r.Body))
	modifiedRequest.ContentLength = int64(len(r.Body))
	modifiedRequest.Host = targetBrokerURL.Host
	modifiedRequest.URL.Path = m[1]

	log.R(r, name).Debugf("Forwarding OSB r to %s", modifiedRequest.URL)

	proxy := httputil.NewSingleHostReverseProxy(targetBrokerURL)
	proxy.Transport = c.fetcher
	recorder := httptest.NewRecorder()

	proxy.ServeHTTP(recorder, modifiedRequest)
	respBody, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		return nil, err
	}

	resp := &web.Response{
		StatusCode: recorder.Code,
		Header:     recorder.HeaderMap,
		Body:       respBody,
	}
	log.R(r, name).Debugf("Service broker replied with status %d", resp.StatusCode)
	return resp, nil
}
