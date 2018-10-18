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

package sm

import (
	"fmt"
	"net/http"

	"time"

	"context"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/pkg/errors"
)

// APIInternalBrokers is the SM API for obtaining the brokers for this proxy
const APIInternalBrokers = "%s/v1/service_brokers"

// Client provides the logic for calling into the Service Manager
//go:generate counterfeiter . Client
type Client interface {
	GetBrokers(ctx context.Context) ([]Broker, error)
}

// ServiceManagerClient allows consuming APIs from a Service Manager
type ServiceManagerClient struct {
	host       string
	httpClient *http.Client
}

// NewClient builds a new Service Manager Client from the provided configuration
func NewClient(config *Settings) (*ServiceManagerClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	httpClient.Timeout = time.Duration(config.RequestTimeout)
	tr := config.Transport

	if tr == nil {
		tr = &SkipSSLTransport{
			SkipSslValidation: config.SkipSSLValidation,
		}
	}

	httpClient.Transport = &BasicAuthTransport{
		Username: config.User,
		Password: config.Password,
		Rt:       tr,
	}

	return &ServiceManagerClient{
		host:       config.URL,
		httpClient: httpClient,
	}, nil
}

// GetBrokers calls the Service Manager in order to obtain all brokers t	hat need to be registered
// in the service broker proxy
func (c *ServiceManagerClient) GetBrokers(ctx context.Context) ([]Broker, error) {
	log.C(ctx).Debugf("Getting brokers for proxy from Service Manager at %s", c.host)
	URL := fmt.Sprintf(APIInternalBrokers, c.host)
	response, err := util.SendRequest(ctx, c.httpClient.Do, http.MethodGet, URL, map[string]string{"catalog": "true"}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}

	list := &Brokers{}
	switch response.StatusCode {
	case http.StatusOK:
		if err = util.BodyToObject(response.Body, list); err != nil {
			return nil, errors.Wrapf(err, "error getting content from body of response with status %s", response.Status)
		}
	default:
		return nil, errors.WithStack(util.HandleResponseError(response))
	}

	return list.Brokers, nil
}
