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

	"github.com/gorilla/mux"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
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
	var err error
	ctx := req.Context()
	defer func() {
		if err != nil {
			util.WriteError(ctx, err, res)
		}
	}()

	var request *web.Request
	req.Body = http.MaxBytesReader(res, req.Body, int64(h.requestBodyMaxSize))
	if request, err = convertToWebRequest(req, res); err != nil {
		return
	}

	var response *web.Response
	response, err = h.Handler.Handle(request)
	ctx = request.Context() // logging filter may have enriched the context with a logger
	if request.IsResponseWriterHijacked() {
		err = nil
		return
	}
	if err != nil {
		return
	}

	// copy response headers
	for k, v := range response.Header {
		if k != "Content-Length" {
			res.Header()[k] = v
		}
	}

	res.WriteHeader(response.StatusCode)
	if _, err = res.Write(response.Body); err != nil {
		// HTTP headers and status are sent already
		// if we return an error, the error Handler will try to send them again
		log.C(ctx).Error("Error sending response", err)
	}
}

func convertToWebRequest(request *http.Request, rw http.ResponseWriter) (*web.Request, error) {
	pathParams := mux.Vars(request)

	var body []byte
	var err error
	if request.Method == "PUT" || request.Method == "POST" || request.Method == "PATCH" {
		body, err = util.RequestBodyToBytes(request)
		if err != nil {
			return nil, isPayloadTooLargeErr(request.Context(), err)
		}
	}

	webReq := &web.Request{
		Request:    request,
		PathParams: pathParams,
		Body:       body,
	}
	webReq.SetResponseWriter(rw)
	return webReq, nil
}

func isPayloadTooLargeErr(ctx context.Context, err error) error {
	// Go http package uses errors.New() to return the below error, so
	// we can only check it with string matching
	if err != nil && err.Error() == "http: request body too large" {
		return &util.HTTPError{
			StatusCode:  http.StatusRequestEntityTooLarge,
			ErrorType:   "PayloadTooLarge",
			Description: "Payload too large",
		}
	}
	return err
}
