/*
 * Copyright 2018 The Service Manager Authors
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

package util_test

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List Utils", func() {
	var (
		requestFunc  func(*http.Request) (*http.Response, error)
		reaction     common.HTTPReaction
		expectations common.HTTPExpectations
		ctx          context.Context
	)
	const URL = "http://some.host/resource"

	BeforeEach(func() {
		ctx = context.TODO()
		reaction = common.HTTPReaction{Status: http.StatusOK}
		expectations = common.HTTPExpectations{URL: URL}
		requestFunc = common.DoHTTP(&reaction, &expectations)
	})

	Describe("ListIterator", func() {
		var it util.ListIterator

		BeforeEach(func() {
			it = util.ListIterator{
				DoRequest: requestFunc,
				URL:       URL,
			}
		})

		It("Returns the items from the response", func() {
			reaction.Body = `{
				"num_items": 2,
				"items":["aaa","bbb"]
			}`
			var items []string
			more, count, err := it.Next(ctx, items, -1)
			Expect(err).To(BeNil())
			Expect(more).To(BeFalse())
			Expect(count).To(Equal(2))
		})
	})

	Describe("ListAll", func() {

	})

})
