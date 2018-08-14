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

	"github.com/Peripli/service-manager/pkg/proxy"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
)

var osbPathPattern = regexp.MustCompile("^" + v1 + root + "/[^/]+(/.*)$")

// Adapter is implemented by OSB handler providers
type Adapter interface {
	Handler() web.HandlerFunc
}

// Controller implements api.Controller by providing OSB API logic
type controller struct {
	adapter Adapter
}

// NewController returns new OSB controller
func NewController(adapter Adapter) web.Controller {
	return &controller{
		adapter: adapter,
	}
}

var _ web.Controller = &controller{}

// BusinessLogic provides handler for the Service Manager OSB business logic
type BusinessLogic struct {
	BrokerStorage storage.Broker
	Encrypter     security.Encrypter
}

// Handler provides implementation to the Adapter interface
// It uses the Reverse proxy implementation to forward the call to the broker
func (a *BusinessLogic) Handler() web.HandlerFunc {
	return func(request *web.Request) (*web.Response, error) {
		logrus.Debug("Executing OSB operation: ", request.URL.Path)
		broker, err := a.fetchBroker(request)
		if err != nil {
			return nil, err
		}
		target, _ := url.Parse(broker.BrokerURL)

		username, password := broker.Credentials.Basic.Username, broker.Credentials.Basic.Password
		plaintextPassword, err := a.Encrypter.Decrypt([]byte(password))
		if err != nil {
			return nil, err
		}

		proxier := proxy.NewReverseProxy(proxy.Options{
			Transport: http.DefaultTransport,
		})
		reqBuilder := proxier.RequestBuilder().Auth(username, string(plaintextPassword))

		m := osbPathPattern.FindStringSubmatch(request.URL.Path)
		if m == nil || len(m) < 2 {
			return nil, fmt.Errorf("could not get OSB path from URL %s", request.URL.Path)
		}
		target.Path = target.Path + m[1]
		reqBuilder.URL(target)

		resp, err := proxier.ProxyRequest(request.Request, reqBuilder, request.Body)
		if err != nil {
			return nil, err
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		webResp := &web.Response{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       respBody,
		}

		logrus.Debugf("Service broker replied with status %d", webResp.StatusCode)
		return webResp, nil
	}
}

func (a *BusinessLogic) fetchBroker(request *web.Request) (*types.Broker, error) {
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

	serviceBroker, err := a.BrokerStorage.Get(brokerID)
	if err != nil {
		logrus.Debugf("Broker with id %s not found in storage during OSB %s operation", brokerID, request.URL.Path)
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	return serviceBroker, nil
}
