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

import "net/url"

type requestBuilder struct {
	username string
	password string
	url      *url.URL
}

// Auth add basic authentication to the request
func (r *requestBuilder) Auth(username, password string) *requestBuilder {
	r.username = username
	r.password = password
	return r
}

// URL which url to forward the new request to
func (r *requestBuilder) URL(url *url.URL) *requestBuilder {
	r.url = url
	return r
}

// RequestBuilder returns new request builder
func (p *Proxy) RequestBuilder() *requestBuilder {
	return &requestBuilder{}
}
