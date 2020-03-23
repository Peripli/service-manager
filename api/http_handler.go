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
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gorilla/mux"
)

// HTTPHandler converts a pkg/web.Handler and pkg/web.HandlerFunc to a standard http.Handler
type HTTPHandler struct {
	Handler            web.Handler
	timeout            time.Duration
	requestBodyMaxSize int
}

// NewHTTPHandler creates a new HTTPHandler from the provided web.Handler
func NewHTTPHandler(handler web.Handler, timeout time.Duration, requestBodyMaxSize int) *HTTPHandler {
	return &HTTPHandler{
		Handler:            handler,
		timeout:            timeout,
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

	var request *web.Request
	req.Body = http.MaxBytesReader(res, req.Body, int64(h.requestBodyMaxSize))
	if request, err = convertToWebRequest(req, res); err != nil {
		util.WriteError(ctx, err, res)
		return
	}

	done := make(chan struct{}, 1)
	panicCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)
	respCh := make(chan *web.Response, 1)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	request.Request = request.WithContext(ctxWithTimeout)

	go func() {
		defer func() {
			// logging filter may have enriched the context with a logger and
			// even in case of panic we want the new context
			ctx = request.Context()
		}()
		defer handlePanic(ctx, panicCh, errCh)

		var response *web.Response
		response, err = h.Handler.Handle(request)
		defer close(done)
		if request.IsResponseWriterHijacked() {
			err = nil
			return
		}
		if err != nil {
			errCh <- err
			return
		}
		if response != nil {
			respCh <- response
		}
	}()

	var response *web.Response
	select {
	case p := <-panicCh:
		panic(p)
	case <-done:
		return
	case err := <-errCh:
		if err != nil {
			util.WriteError(ctx, err, res)
		}
		return
	case <-ctxWithTimeout.Done():
		// fmt.Println(">>>>>>here1")
		select {
		case err := <-errCh:
			// fmt.Println(">>>>>>here2")
			util.WriteError(ctx, err, res)
			return
		case response = <-respCh:
			fmt.Println(">>>>>Whyy??", response.StatusCode)
		}
	case response = <-respCh:
	}

	// copy response headers
	for k, v := range response.Header {
		if k != "Content-Length" {
			res.Header()[k] = v
		}
	}

	defer func() {
		if err != nil {
			util.WriteError(ctx, err, res)
		}
	}()

	res.WriteHeader(response.StatusCode)
	if _, err = res.Write(response.Body); err != nil {
		// HTTP headers and status are sent already
		// if we return an error, the error Handler will try to send them again
		log.C(ctx).Error("Error sending response", err)
	}
}

func handlePanic(ctx context.Context, panicCh chan interface{}, errCh chan error) {
	if e := recover(); e != nil {
		err, ok := e.(error)
		if ok && err == http.ErrAbortHandler {
			errCh <- err
		} else if httpErr, isHTTPErr := err.(*util.HTTPError); isHTTPErr {
			errCh <- httpErr
		} else {
			panicCh <- e
		}
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
