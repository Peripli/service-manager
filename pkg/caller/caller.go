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

package caller

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Name                 string
	Retries              int
	FallbackHandler      func(error) error
	RoundTripper         http.RoundTripper
	IsCallSuccessfulFunc func(response *web.Response) bool
}

func DefaultConfig(name string) Config {
	return Config{
		Name:    name,
		Retries: 1,
		FallbackHandler: func(err error) error {
			return err
		},
		RoundTripper: http.DefaultTransport,
		IsCallSuccessfulFunc: func(response *web.Response) bool {
			return response.StatusCode < 500
		},
	}
}

type caller struct {
	Name                 string
	Retries              int
	FallbackHandler      func(error) error
	IsCallSuccessfulFunc func(response *web.Response) bool
	RoundTripper         http.RoundTripper
}

func New(config Config) (*caller, error) {
	return &caller{
		Name:                 config.Name,
		Retries:              config.Retries,
		FallbackHandler:      config.FallbackHandler,
		RoundTripper:         config.RoundTripper,
		IsCallSuccessfulFunc: config.IsCallSuccessfulFunc,
	}, nil
}

func (c *caller) Call(r *web.Request) (*web.Response, error) {
	output := make(chan *web.Response)
	errors := hystrix.Go(c.Name, func() error {
		return c.callWithRetries(r, output)
	}, func(e error) error {
		if c.FallbackHandler != nil {
			return c.FallbackHandler(e)
		}
		logrus.Errorf("Failed to execute request from %s. Error: %s", c.Name, e)
		return e
	})
	select {
	case result := <-output:
		return result, nil
	case err := <-errors:
		return nil, err
	}
}

func (c *caller) callWithRetries(r *web.Request, output chan *web.Response) error {
	retries := 0
	for {
		response, err := c.makeCall(r)
		if err != nil || !c.IsCallSuccessfulFunc(response) {
			time.Sleep(100 * time.Millisecond)
			retries ++
			if retries >= c.Retries {
				logrus.Debugf("Call failed after %d retries", retries)
				return err
			}
			logrus.Debugf("Call failed. Retrying...")
		} else {
			logrus.Debugf("Call succeeded at retry #%d", retries)
			output <- response
			return nil
		}
	}
}

func (c *caller) makeCall(r *web.Request) (*web.Response, error) {
	response, err := c.RoundTripper.RoundTrip(r.Request)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return &web.Response{
		StatusCode: response.StatusCode,
		Body:       body,
		Header:     response.Header,
	}, nil
}
