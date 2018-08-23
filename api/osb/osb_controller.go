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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"

	"net/http/httputil"

	"net/http/httptest"

	"bytes"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

var osbPathPattern = regexp.MustCompile("^" + web.OSBURL + "/[^/]+(/.*)$")

// BrokerRoundTripper is implemented by OSB handler providers
type BrokerRoundTripper interface {
	http.RoundTripper

	Broker(brokerID string) (*types.Broker, error)
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

func (c *controller) handler(request *web.Request) (*web.Response, error) {
	logrus.Debug("Executing OSB operation: ", request.URL.Path)

	brokerID, ok := request.PathParams[BrokerIDPathParam]
	if !ok {
		logrus.Debugf("error creating OSB client: brokerID path parameter not found")
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "invalid broker id path parameter",
			StatusCode:  http.StatusBadRequest,
		}
	}
	logrus.Debugf("Obtained path parameter [brokerID = %s] from path params", brokerID)

	broker, err := c.fetcher.Broker(brokerID)
	if err != nil {
		return nil, err
	}

	targetBrokerURL, _ := url.Parse(broker.BrokerURL)

	m := osbPathPattern.FindStringSubmatch(request.URL.Path)
	if m == nil || len(m) < 2 {
		return nil, fmt.Errorf("could not get OSB path from URL %s", request.URL.Path)
	}

	modifiedRequest := request.Request.WithContext(request.Context())
	modifiedRequest.SetBasicAuth(broker.Credentials.Basic.Username, broker.Credentials.Basic.Password)
	modifiedRequest.Body = ioutil.NopCloser(bytes.NewReader(request.Body))
	modifiedRequest.ContentLength = int64(len(request.Body))
	modifiedRequest.Host = targetBrokerURL.Host
	modifiedRequest.URL.Path = m[1]

	logrus.Debugf("Forwarding OSB request to %s", modifiedRequest.URL)

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
	logrus.Debugf("Service broker replied with status %d", resp.StatusCode)
	return resp, nil
}
