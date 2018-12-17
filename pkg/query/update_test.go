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

package query

import (
	"fmt"

	"github.com/tidwall/sjson"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Update", func() {

	var body []byte

	Describe("Label changes for body", func() {
		BeforeEach(func() {
			body = []byte(fmt.Sprintf(`{"labels":[
	{
		"op": "%s",
		"key": "key1",
		"values": ["val1", "val2"]
	}
]}`, AddLabelOperation))
		})

		Context("When label has values", func() {
			It("Should be ok", func() {
				changes, err := LabelChangesFromJSON(body)
				Expect(err).ToNot(HaveOccurred())
				Expect(changes).To(ConsistOf(&LabelChange{Operation: AddLabelOperation, Key: "key1", Values: []string{"val1", "val2"}}))
			})
		})

		Context("When label has no values", func() {
			It("Should return error", func() {
				body, err := sjson.DeleteBytes(body, "labels.0.values")
				Expect(err).ToNot(HaveOccurred())
				changes, err := LabelChangesFromJSON(body)
				Expect(err).To(HaveOccurred())
				Expect(changes).To(BeNil())
			})
		})

		Context("When there are no labels in the body", func() {
			It("Should return no label changes", func() {
				body, err := sjson.DeleteBytes(body, "labels")
				Expect(err).ToNot(HaveOccurred())
				changes, err := LabelChangesFromJSON(body)
				Expect(err).ToNot(HaveOccurred())
				Expect(changes).To(BeEmpty())
			})
		})

		Context("When labels are not a valid structure", func() {
			It("Should return error", func() {
				body, err := sjson.SetBytes(body, "labels.0.values", "not-array")
				Expect(err).ToNot(HaveOccurred())
				changes, err := LabelChangesFromJSON(body)
				Expect(err).To(HaveOccurred())
				Expect(changes).To(BeNil())
			})
		})
	})
})
