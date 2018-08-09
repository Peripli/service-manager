package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

type myProxy struct {
	reverseProxy *httputil.ReverseProxy
}

type requestBuilder struct {
	username string
	password string
	url      *url.URL
}

func (r *requestBuilder) Auth(username, password string) *requestBuilder {
	r.username = username
	r.password = password
	return r
}

func (r *requestBuilder) URL(url *url.URL) *requestBuilder {
	r.url = url
	return r
}

func (p *myProxy) RequestBuilder() *requestBuilder {
	return &requestBuilder{}
}

func (p *myProxy) ProxyRequest(req *http.Request, reqBuilder *requestBuilder, body []byte) (*web.Response, error) {
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

	responseBody, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		return nil, err
	}

	headers := recorder.HeaderMap
	resp := &web.Response{
		StatusCode: recorder.Code,
		Body:       responseBody,
		Header:     headers,
	}
	return resp, nil
}

func ReverseProxy() *myProxy {
	return &myProxy{
		reverseProxy: &httputil.ReverseProxy{
			Director: func(req *http.Request) {

			},
		},
	}
}
