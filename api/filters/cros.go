package filters

import (
	"bytes"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

// ResponseHeaderStripperFilter is a web.Filter used to strip headers from all OSB calls
// on their way back to a platform
type CROSFilter struct {
	Environment env.Environment
	web.Filter
	Headers []string
}

// Name implements web.Named and returns the Filter name
func (f *CROSFilter) Name() string {
	return "CROSFilter"
}

func resWrap(res *web.Response, err error, allowedUrl string) (*web.Response, error) {
	crosHeader := res.Header.Get("Access-Control-Allow-Origin")
	if crosHeader!= "*"{
		res.Header.Set("Access-Control-Allow-Origin", allowedUrl)
	}
	return res, err
}

// Run implements web.Filter and represents the Response Header Stripper middleware function that
// strips blacklisted headers
func (f *CROSFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	var allowedUrl string
	reqHost := request.Header.Get("Origin")
	allowedHost := f.Environment.Get("cross.allowed_host")
	if allowedHost != nil {
		allowedUrl = allowedHost.(string)
	}

	var webRes web.Response
	webRes.StatusCode = 200
	webRes.Header = http.Header{}
	webRes.Header.Add("Access-Control-Allow-Origin", allowedUrl)
	if request.Method == "OPTIONS" {
		if allowedHost == reqHost {
			webRes.Header.Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			webRes.Header.Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
			return &webRes, nil
		} else {
			return &webRes, nil
		}
	} else if allowedHost == reqHost {
		res, err := next.Handle(request)
		if err == nil {
			return resWrap(res, err, allowedUrl)
		} else {
			stream := bytes.NewBufferString("")
			webRes.Header.Add("Content-Type", "application/json")
			httpError := util.ToHTTPError(request.Context(), err)
			webRes.StatusCode = httpError.StatusCode
			encoder := json.NewEncoder(stream)
			err := encoder.Encode(httpError)
			if err != nil {
				return &webRes, err
			}
			webRes.Body = stream.Bytes()
			return &webRes, nil
		}
	}

	return next.Handle(request)
}

// FilterMatchers implements the web.Filter interface and returns the conditions
// on which the filter should be executed
func (f *CROSFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("*/**"),
				web.Methods(http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete),
			},
		},
	}
}
