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

package web_test

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

var _ = Describe("Web test", func() {

	Describe("NewJSONResponse", func() {
		const testTypeValid = `{"field1":"Value1", "field2":"Value2"}`

		It("builds a web.Response containing the marshalled value with a Content-Type header", func() {
			expectedCode := http.StatusOK
			testValue := testTypeValidated{
				Field1: "Value1",
				Field2: "Value2",
			}
			response, err := web.NewJSONResponse(expectedCode, testValue)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(expectedCode))
			Expect(response.Body).Should(MatchJSON(testTypeValid))
			Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
		})

		It("builds a web.Response containing an empty response value when return code is 204 No Content", func() {
			testJSONResponse(http.StatusNoContent, web.EmptyResponseBody{}, []byte{})
		})

		It("builds a web.Response containing an empty response value when return code is 200 OK and EmptyResponseBody is provided", func() {
			testJSONResponse(http.StatusOK, web.EmptyResponseBody{}, []byte{})
		})

		It("builds a web.Response containing a non-empty response value when instead of EmptyResponseBody an empty data structure is provided", func() {
			response, err := web.NewJSONResponse(http.StatusOK, struct{}{})

			Expect(err).ShouldNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
			Expect(response.Body).ShouldNot(BeEmpty())
			Expect(response.Body).ShouldNot(Equal(web.EmptyResponseBody{}))
			Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
		})
	})
})

type testTypeValidated struct {
	Field1 string `json:"field1"`
	Field2 string `json:"field2"`
}

func (tt testTypeValidated) Validate() error {
	if tt.Field1 == "" {
		return fmt.Errorf("empty field1")
	}
	if tt.Field2 == "" {
		return fmt.Errorf("empty field2")
	}
	return nil
}

func testJSONResponse(expectedCode int, providedBody, expectedMarshalledBody interface{}) {
	response, err := web.NewJSONResponse(expectedCode, providedBody)

	Expect(err).ShouldNot(HaveOccurred())
	Expect(response.StatusCode).To(Equal(expectedCode))
	Expect(response.Body).Should(Equal(expectedMarshalledBody))
	Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
}
