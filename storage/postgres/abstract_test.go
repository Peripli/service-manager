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

package postgres

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Postgres Storage Abstract", func() {
	Describe("updateQuery", func() {

		Context("Called with structure with no db tag", func() {
			It("Should return proper query", func() {
				type ts struct {
					Field string
				}
				query := updateQuery("n/a", ts{Field: "value"})
				Expect(query).To(Equal("UPDATE n/a SET field = :field WHERE id = :id"))
			})
		})

		Context("Called with structure with db tag", func() {
			It("Should return proper query", func() {
				type ts struct {
					Field string `db:"taggedField"`
				}
				query := updateQuery("n/a", ts{Field: "value"})
				Expect(query).To(Equal("UPDATE n/a SET taggedField = :taggedField WHERE id = :id"))
			})
		})

		Context("Called with structure with empty field", func() {
			It("allows setting default values for fields", func() {
				type ts struct {
					Field string
				}
				query := updateQuery("n/a", ts{})
				Expect(query).To(Equal("UPDATE n/a SET field = :field WHERE id = :id"))
			})
		})

		Context("Called with structure with nil field", func() {
			It("ignores nils", func() {
				type ts struct {
					Field *string
				}
				query := updateQuery("n/a", ts{})
				Expect(query).To(Equal(""))
			})
		})

		Context("Called with structure with no fields", func() {
			It("Should return proper query", func() {
				type ts struct{}
				query := updateQuery("n/a", ts{})
				Expect(query).To(Equal(""))
			})
		})
	})
})
