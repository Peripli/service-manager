/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"net/http"
)

type BrokerClient struct {
	tlsConfig               *tls.Config
	broker                  *types.ServiceBroker
	requestHandlerDecorated util.DoRequestFunc
}

func NewBrokerClient(broker *types.ServiceBroker, requestHandler util.DoRequestWithClientFunc,  ctx context.Context) (*BrokerClient, error) {
	tlsConfig, err := broker.GetTLSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get client for broker %s: %v", broker.Name, err)
	}

	if requestHandler == nil {
		return nil, errors.New("a request handler func is required")
	}

	bc := &BrokerClient{}
	bc.tlsConfig = tlsConfig
	bc.broker = broker
	bc.requestHandlerDecorated = bc.authAndTlsDecorator(requestHandler)
	return bc, nil
}

func (bc *BrokerClient) addBasicAuth(req *http.Request) *BrokerClient {
	req.SetBasicAuth(bc.broker.Credentials.Basic.Username, bc.broker.Credentials.Basic.Password)
	return bc
}

func (bc *BrokerClient) authAndTlsDecorator(requestHandler util.DoRequestWithClientFunc) util.DoRequestFunc {
	return func(req *http.Request) (*http.Response, error) {
		client := http.DefaultClient
		ctx := req.Context()
		logger := log.C(ctx)
		if bc.broker.Credentials.Basic != nil && bc.broker.Credentials.Basic.Username != "" && bc.broker.Credentials.Basic.Password != "" {
			bc.addBasicAuth(req)
		}

		if bc.tlsConfig != nil {
			client = &http.Client{}
			logger.Infof("configuring broker tls for %s", bc.broker.Name)
			client.Transport = GetTransportWithTLS(bc.tlsConfig)
			return requestHandler(req, client)
		}

		return requestHandler(req, client)
	}
}

func (bc *BrokerClient) SendRequest(ctx context.Context, method, url string, params map[string]string, body interface{}, headers map[string]string) (*http.Response, error) {
	return util.SendRequestWithHeaders(ctx, bc.requestHandlerDecorated, method, url, params, body, headers)
}
