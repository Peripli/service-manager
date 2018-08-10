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

package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"

	"github.com/sirupsen/logrus"
)

type Proxy struct {
	reverseProxy *httputil.ReverseProxy
}

type Options struct {
	Transport http.RoundTripper
}

func (p *Proxy) ProxyRequest(req *http.Request, reqBuilder *requestBuilder, body []byte) (*http.Response, error) {
	modifiedRequest := req.WithContext(req.Context())

	if reqBuilder.username != "" && reqBuilder.password != "" {
		modifiedRequest.SetBasicAuth(reqBuilder.username, reqBuilder.password)
	}

	modifiedRequest.Host = reqBuilder.url.Host
	modifiedRequest.URL.Scheme = reqBuilder.url.Scheme
	modifiedRequest.URL.Host = reqBuilder.url.Host
	modifiedRequest.URL.Path = reqBuilder.url.Path

	modifiedRequest.Body = ioutil.NopCloser(bytes.NewReader(body))
	modifiedRequest.ContentLength = int64(len(body))

	logrus.Debugf("Forwarding request to %s", modifiedRequest.URL)

	recorder := httptest.NewRecorder()
	p.reverseProxy.ServeHTTP(recorder, modifiedRequest)

	headers := recorder.HeaderMap
	resp := &http.Response{
		StatusCode: recorder.Code,
		Body:       ioutil.NopCloser(recorder.Body),
		Header:     headers,
	}

	return resp, nil
}

func ReverseProxy(options Options) *Proxy {
	return &Proxy{
		reverseProxy: &httputil.ReverseProxy{
			Transport: options.Transport,
			Director:  func(req *http.Request) {},
		},
	}
}
