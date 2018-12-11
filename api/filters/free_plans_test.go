
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
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
	"net/http"
)

// use mock db or directly use fakes / stubs and test corner cases
var _ = Describe("Free Plans Filter", func() {
	var freePlansFilter := &FreeServicePlansFilter{
		Repository: nil,
	}

	var request *web.Request
	var handler *webfakes.FakeHandler

	BeforeEach(func() {
		request = &web.Request{Request: &http.Request{}}
		request.Header = http.Header{}
		handler = &webfakes.FakeHandler{}
	})

})
