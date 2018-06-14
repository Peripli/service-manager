package server

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/types"
	"github.com/gorilla/mux"
)

func newHttpHandler(filters []filter.Filter, handler filter.Handler) http.Handler {
	return httpHandler(chain(filters, handler))
}

type httpHandler filter.Handler

func (h httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if err := h.serve(res, req); err != nil {
		rest.HandleError(err, res)
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

	code := restRes.StatusCode
	if code == 0 {
		code = http.StatusOK
	}
	res.WriteHeader(code)
	_, err = res.Write(restRes.Body)
	if err != nil {
		logrus.Error("Error sending response", err)
	}
	return nil
}

func readRequest(request *http.Request) (*filter.Request, error) {
	pathParams := mux.Vars(request)

	var body []byte
	var err error
	if request.Method == "PUT" || request.Method == "POST" {
		body, err = readBody(request)
	}

	return &filter.Request{
		Request:    request,
		PathParams: pathParams,
		Body:       body,
	}, err
}

func readBody(request *http.Request) ([]byte, error) {
	contentType := request.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, types.NewErrorResponse(errors.New("Invalid media type provided"),
			http.StatusUnsupportedMediaType, "InvalidMediaType")
	}
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		return nil, types.NewErrorResponse(errors.New("Request body is not valid JSON"),
			http.StatusBadRequest, "BadRequest")
	}

	return body, nil
}

func chain(filters []filter.Filter, handler filter.Handler) filter.Handler {
	if len(filters) == 0 {
		return handler
	}
	next := chain(filters[1:], handler)
	f := filters[0].Middleware
	return func(req *filter.Request) (*filter.Response, error) {
		return f(req, next)
	}
}

func matchFilters(endpoint *rest.Endpoint, filters []filter.Filter) []filter.Filter {
	matches := []filter.Filter{}
	for _, filter := range filters {
		if matchPath(endpoint.Path, filter.PathPattern) &&
			matchMethod(endpoint.Method, filter.Methods) {
			matches = append(matches, filter)
		}
	}
	logrus.Debugf("%d filters for endpoint %v", len(matches), endpoint)
	return matches
}

func matchPath(endpointPath string, pattern string) bool {
	if pattern == "" {
		return true
	}
	// TODO: add support for **
	match, err := path.Match(pattern, endpointPath)
	if err != nil {
		logrus.Fatalf("Invalid endpoint path pattern %s: %v", endpointPath, err)
	}
	return match
}

func matchMethod(method string, methods []string) bool {
	if method == rest.AllMethods || methods == nil {
		return true
	}
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}
