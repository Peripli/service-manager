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

// Controller implements api.Controller by providing OSB API logic
type Controller struct {
	BrokerStorage storage.Broker
	Filters       web.Filters
	Encrypter     security.Encrypter
	Handler       web.HandlerFunc
}

var _ web.Controller = &Controller{}

func (c *Controller) handler(request *web.Request) (*web.Response, error) {
	logrus.Debug("Executing OSB operation: ", request.URL.Path)
	broker, err := c.fetchBroker(request)
	if err != nil {
		return nil, err
	}
	target, _ := url.Parse(broker.BrokerURL)

	username, password := broker.Credentials.Basic.Username, broker.Credentials.Basic.Password
	plaintextPassword, err := c.Encrypter.Decrypt([]byte(password))
	if err != nil {
		return nil, err
	}

	proxier := proxy.ReverseProxy()
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
	logrus.Debugf("Service broker replied with status %d", resp.StatusCode)
	return resp, nil
}

func (c *Controller) fetchBroker(request *web.Request) (*types.Broker, error) {
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

	serviceBroker, err := c.BrokerStorage.Get(brokerID)
	if err != nil {
		logrus.Debugf("Broker with id %s not found in storage during OSB %s operation", brokerID, request.URL.Path)
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	return serviceBroker, nil
}
