/*
 *    Copyright 2018 The Service Manager Authors
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

package rest

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gorilla/mux"
)

type httpHandler web.Handler

// NewHTTPHandler wraps the controller handler and its filters in a http.Handler function
func NewHTTPHandler(filters []web.Filter, handler web.Handler) http.Handler {
	return httpHandler(chain(filters, handler))
}

func (h httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if err := h.serve(res, req); err != nil {
		HandleError(err, res)
	}
}

func (h httpHandler) serve(res http.ResponseWriter, req *http.Request) error {
	restReq, err := readRequest(req)
	if err != nil {
		return err
	}

	restRes, err := h(restReq)
	if err != nil {
		return err
	}

	// copy response headers
	for k, v := range restRes.Header {
		if k != "Content-Length" {
			res.Header()[k] = v
		}
	}

	res.WriteHeader(restRes.StatusCode)
	_, err = res.Write(restRes.Body)
	if err != nil {
		// HTTP headers and status are sent already
		// if we return an error, the error handler will try to send them again
		logrus.Error("Error sending response", err)
	}
	return nil
}

func readRequest(request *http.Request) (*web.Request, error) {
	pathParams := mux.Vars(request)

	var body []byte
	var err error
	if request.Method == "PUT" || request.Method == "POST" || request.Method == "PATCH" {
		body, err = readBody(request)
	}

	return &web.Request{
		Request:    request,
		PathParams: pathParams,
		Body:       body,
	}, err
}

func readBody(request *http.Request) ([]byte, error) {
	contentType := request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, web.NewHTTPError(errors.New("Invalid media type provided"),
			http.StatusUnsupportedMediaType, "InvalidMediaType")
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		return nil, web.NewHTTPError(errors.New("Request body is not valid JSON"),
			http.StatusBadRequest, "BadRequest")
	}

	return body, nil
}

func chain(filters []web.Filter, handler web.Handler) web.Handler {
	if len(filters) == 0 {
		return handler
	}
	next := chain(filters[1:], handler)
	f := filters[0].Middleware
	if f == nil {
		logrus.Panicf("Missing middleware function for filter %s", filters[0].Name)
	}
	return func(req *web.Request) (*web.Response, error) {
		return f(req, next)
	}
}
