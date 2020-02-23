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
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("List Utils", func() {
	var (
		requestFunc  func(*http.Request, *http.Client) (*http.Response, error)
		reaction     common.HTTPReaction
		expectations common.HTTPExpectations
		ctx          context.Context
	)
	const url = "http://some.host/resource"

	BeforeEach(func() {
		ctx = context.TODO()
		reaction = common.HTTPReaction{Status: http.StatusOK}
		expectations = common.HTTPExpectations{
			URL:    url,
			Params: map[string]string{},
		}
		requestFunc = common.DoHTTP(&reaction, &expectations)
	})

	Describe("ListIterator", func() {
		var it util.ListIterator

		BeforeEach(func() {
			it = util.ListIterator{
				DoRequest: requestFunc,
				URL:       url,
			}
		})

		When("items is not a pointer to a slice", func() {
			It("Returns an error", func() {
				var items string
				_, _, err := it.Next(ctx, &items, -1)
				Expect(err.Error()).To(ContainSubstring("*string"))
			})
		})

		When("Requesting only the count", func() {
			It("Returns only the count", func() {
				reaction.Body = `{
					"num_items": 2,
					"items":["aaa","bbb"]
				}`
				more, count, err := it.Next(ctx, nil, -1)
				Expect(err).To(BeNil())
				Expect(more).To(BeFalse())
				Expect(count).To(Equal(int64(2)))
			})
		})

		When("There is only one page of items", func() {
			It("Returns the items from the response", func() {
				reaction.Body = `{
					"num_items": 2,
					"items":["aaa","bbb"]
				}`
				var items []string
				more, count, err := it.Next(ctx, &items, -1)
				Expect(err).To(BeNil())
				Expect(more).To(BeFalse())
				Expect(count).To(Equal(int64(2)))
				Expect(items).To(Equal([]string{"aaa", "bbb"}))
			})
		})

		When("There are multiple pages of items", func() {
			It("Requests next page with proper token", func() {
				var items []string

				By("Requesting the first page")
				reaction.Body = `{
					"num_items": 3,
					"token": "page2",
					"items":["aaa","bbb"]
				}`
				more, count, err := it.Next(ctx, &items, -1)
				Expect(err).To(BeNil())
				Expect(more).To(BeTrue())
				Expect(count).To(Equal(int64(3)))

				By("Requesting the next page - the last one")
				expectations.Params["token"] = "page2"
				reaction.Body = `{
					"num_items": 3,
					"items":["ccc"]
				}`
				more, count, err = it.Next(ctx, &items, -1)
				Expect(err).To(BeNil())
				Expect(more).To(BeFalse())
				Expect(count).To(Equal(int64(3)))

				By("Requesting the next page after the last one")
				_, _, err = it.Next(ctx, &items, -1)
				Expect(err.Error()).To(Equal("iteration already complete"))
			})
		})

		When("The page size is provided", func() {
			It("Sets the max_items parameter", func() {
				expectations.Params["max_items"] = "5"
				reaction.Body = `{
					"num_items": 2,
					"items":["aaa","bbb"]
				}`
				var items []string
				_, _, err := it.Next(ctx, &items, 5)
				Expect(err).To(BeNil())
			})
		})

		When("There is a network error", func() {
			It("Returns an error", func() {
				reaction.Err = errors.New("Network error")
				_, _, err := it.Next(ctx, nil, -1)
				Expect(err.Error()).To(ContainSubstring("Network error"))
				Expect(err.Error()).To(ContainSubstring("GET " + url))
			})
		})

		When("There the server returns error status", func() {
			It("Returns an error", func() {
				reaction.Status = 500
				_, _, err := it.Next(ctx, nil, -1)
				Expect(err.Error()).To(ContainSubstring("500"))
				Expect(err.Error()).To(ContainSubstring("GET " + url))
			})
		})

		When("There the response body is invalid JSON", func() {
			It("Returns an error", func() {
				reaction.Body = `{`
				_, _, err := it.Next(ctx, nil, -1)
				Expect(err.Error()).To(ContainSubstring("error parsing response body"))
				Expect(err.Error()).To(ContainSubstring("GET " + url))
			})
		})
	})

	Describe("ListAll", func() {
		When("items is not a pointer to a slice", func() {
			It("Returns an error", func() {
				var items string
				err := util.ListAll(ctx, requestFunc, url, &items)
				Expect(err.Error()).To(ContainSubstring("items should be a pointer to a slice"))
			})
		})

		When("There the server returns error status", func() {
			It("Returns an error", func() {
				reaction.Status = 500
				var items []string
				err := util.ListAll(ctx, requestFunc, url, &items)
				Expect(err.Error()).To(ContainSubstring("500"))
				Expect(err.Error()).To(ContainSubstring("GET " + url))
			})
		})

		When("There is a single page", func() {
			It("Returns all the items", func() {
				reaction.Body = `{
					"num_items": 2,
					"items":["aaa","bbb"]
				}`
				var items []string
				err := util.ListAll(ctx, requestFunc, url, &items)
				Expect(err).To(BeNil())
				Expect(items).To(Equal([]string{"aaa", "bbb"}))
			})
		})

		When("There are multiple pages", func() {
			It("Returns all the items", func() {
				sequence := []common.HTTPCouple{
					{
						Reaction: &common.HTTPReaction{
							Status: http.StatusOK,
							Body: `{
								"num_items": 3,
								"token": "page2",
								"items":["aaa","bbb"]
							}`,
						},
					},
					{
						Reaction: &common.HTTPReaction{
							Status: http.StatusOK,
							Body: `{
								"num_items": 3,
								"items":["ccc"]
							}`,
						},
					},
				}
				var items []string
				err := util.ListAll(ctx, common.DoHTTPSequence(sequence), url, &items)
				Expect(err).To(BeNil())
				Expect(items).To(Equal([]string{"aaa", "bbb", "ccc"}))
			})
		})
	})

})
