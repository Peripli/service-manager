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

package api

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gorilla/mux"
)

// HTTPHandler converts a pkg/web.Handler and pkg/web.HandlerFunc to a standard http.Handler
type HTTPHandler struct {
	Handler            web.Handler
	requestBodyMaxSize int
}

// NewHTTPHandler creates a new HTTPHandler from the provided web.Handler
func NewHTTPHandler(handler web.Handler, requestBodyMaxSize int) *HTTPHandler {
	return &HTTPHandler{
		Handler:            handler,
		requestBodyMaxSize: requestBodyMaxSize,
	}
}

// Handle implements the web.Handler interface
func (h *HTTPHandler) Handle(req *web.Request) (resp *web.Response, err error) {
	return h.Handler.Handle(req)
}

// ServeHTTP implements the http.Handler interface and allows wrapping web.Handlers into http.Handlers
func (h *HTTPHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if err := h.serve(res, req); err != nil {
		util.WriteError(err, res)
	}
}

func (h *HTTPHandler) serve(res http.ResponseWriter, req *http.Request) error {
	req.Body = http.MaxBytesReader(res, req.Body, int64(h.requestBodyMaxSize))

	request, err := convertToWebRequest(req, res)
	if err != nil {
		return err
	}

	response, err := h.Handler.Handle(request)
	if request.IsResponseWriterHijacked() {
		return nil
	}
	if err != nil {
		return err
	}

	// copy response headers
	for k, v := range response.Header {
		if k != "Content-Length" {
			res.Header()[k] = v
		}
	}

	res.WriteHeader(response.StatusCode)
	_, err = res.Write(response.Body)
	if err != nil {
		// HTTP headers and status are sent already
		// if we return an error, the error Handler will try to send them again
		log.C(req.Context()).Error("Error sending response", err)
	}
	return nil
}

func convertToWebRequest(request *http.Request, rw http.ResponseWriter) (*web.Request, error) {
	pathParams := mux.Vars(request)

	var body []byte
	var err error
	if request.Method == "PUT" || request.Method == "POST" || request.Method == "PATCH" {
		body, err = util.RequestBodyToBytes(request)
		err = isPayloadTooLargeErr(request.Context(), err)
	}

	webReq := &web.Request{
		Request:    request,
		PathParams: pathParams,
		Body:       body,
	}
	webReq.SetResponseWriter(rw)
	return webReq, err
}

func isPayloadTooLargeErr(ctx context.Context, err error) error {
	// Go http package uses errors.New() to return the below error, so
	// we can only check it with string matching
	if err != nil && err.Error() == "http: request body too large" {
		log.C(ctx).Errorf(err.Error())
		return &util.HTTPError{
			StatusCode:  http.StatusRequestEntityTooLarge,
			ErrorType:   "PayloadTooLarge",
			Description: "Payload too large",
		}
	}
	return err
}
