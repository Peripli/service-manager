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

package filters

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
	"strconv"
)

// PagingFilterName is the name of the paging filter
const PagingFilterName = "PagingFilter"

// pagingFilter is filter that adds paging criteria to request
type pagingFilter struct {
	DefaultPageSize int
	MaxPageSize     int
}

// NewPagingFilter returns new paging filter for given default and max page sizes
func NewPagingFilter(defaultPageSize, maxPageSize int) web.Filter {
	return &pagingFilter{
		DefaultPageSize: defaultPageSize,
		MaxPageSize:     maxPageSize,
	}
}

// Name returns the name of the paging filter
func (*pagingFilter) Name() string {
	return PagingFilterName
}

// Run represents the paging middleware function that processes the request and configures needed criteria.
func (pf *pagingFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	maxItems := req.URL.Query().Get("max_items")
	limit, err := pf.parseMaxItemsQuery(maxItems)
	if err != nil {
		return nil, err
	}

	rawToken := req.URL.Query().Get("token")
	token, err := pf.parsePageToken(rawToken)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, "limit", limit)
	ctx = context.WithValue(ctx, "user_provided_query", query.CriteriaForContext(ctx))
	if limit > 0 {
		ctx, err = query.AddCriteria(ctx, query.LimitResultBy(limit+1),
			query.OrderResultBy("paging_sequence", query.AscOrder),
			query.ByField(query.GreaterThanOperator, "paging_sequence", strconv.Itoa(token)))
		if err != nil {
			return nil, err
		}
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

func (pf *pagingFilter) parseMaxItemsQuery(maxItems string) (int, error) {
	limit := pf.DefaultPageSize
	var err error
	if maxItems != "" {
		limit, err = strconv.Atoi(maxItems)
		if err != nil {
			return -1, &util.HTTPError{
				ErrorType:   "InvalidMaxItems",
				Description: fmt.Sprintf("max_items should be integer: %v", err),
				StatusCode:  http.StatusBadRequest,
			}
		}
		if limit < 0 {
			return -1, &util.HTTPError{
				ErrorType:   "InvalidMaxItems",
				Description: fmt.Sprintf("max_items cannot be negative"),
				StatusCode:  http.StatusBadRequest,
			}
		}
		if limit > pf.MaxPageSize {
			limit = pf.MaxPageSize
		}
	}
	return limit, nil
}

func (pf *pagingFilter) parsePageToken(token string) (int, error) {
	var targetPageSequence int
	if token != "" {
		base64DecodedTokenBytes, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return 0, &util.HTTPError{
				ErrorType:   "TokenInvalid",
				Description: fmt.Sprintf("Invalid token provided: %v", err),
				StatusCode:  http.StatusNotFound,
			}
		}
		base64DecodedToken := string(base64DecodedTokenBytes)
		targetPageSequence, err = strconv.Atoi(base64DecodedToken)
		if err != nil {
			return 0, &util.HTTPError{
				ErrorType:   "TokenInvalid",
				Description: fmt.Sprintf("Invalid token provided: %v", err),
				StatusCode:  http.StatusNotFound,
			}
		}
	}
	return targetPageSequence, nil
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed.
func (*pagingFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.PlatformsURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.ServiceBrokersURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.ServiceOfferingsURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.ServicePlansURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.VisibilitiesURL + "/**"),
			},
		},
	}
}
