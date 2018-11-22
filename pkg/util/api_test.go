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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/Peripli/service-manager/pkg/util"
	. "github.com/onsi/gomega"
)

func TestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Util test suite")
}

func validateHTTPErrorOccured(err error, expectedStatusCode int) {
	Expect(err).Should(HaveOccurred())

	httpError, ok := err.(*util.HTTPError)
	Expect(ok).To(BeTrue())

	Expect(httpError.StatusCode).To(Equal(expectedStatusCode))

	Expect(httpError.ErrorType).To(Not(BeEmpty()))
	Expect(httpError.Description).To(Not(BeEmpty()))
}

var _ = Describe("Utils test", func() {

	Describe("HasRFC3986ReservedSymbols", func() {

		assertHasReservedCharacters := func(input string) {
			It("should return true", func() {
				Expect(util.HasRFC3986ReservedSymbols(input)).To(Equal(true))
			})
		}

		assertNoReservedCharacters := func(input string) {
			It("should return false", func() {
				Expect(util.HasRFC3986ReservedSymbols(input)).To(Equal(false))
			})
		}

		assertReservedCases := func(cases []string, hasReserved bool) {
			for _, str := range cases {
				if hasReserved {
					assertHasReservedCharacters(str)
				} else {
					assertNoReservedCharacters(str)
				}
			}
		}

		Context("HasRFC3986ReservedSymbols with single characters", func() {
			reserved := []string{":", "/", "?", "#", "[", "]", "@", "!", "$", "&", "'", "(", ")", "*", "+", ",", ";", "="}
			assertReservedCases(reserved, true)
		})

		Context("HasRFC3986ReservedSymbols with multiple symbols", func() {
			cases := []string{"@a\\/", "@a@", "a:b", "a:;b", ":;@", "()", "+a+", "[a+]", "a=3?"}
			assertReservedCases(cases, true)
		})

		Context("HasRFC3986ReservedSymbols with no reserved symbols", func() {
			cases := []string{"a", "a~b", "a_b", "a-b", "", "74a", "a00", "--a", "-a", "a-", "a--", "-"}
			assertReservedCases(cases, false)
		})
	})

	Describe("RequestBodyToBytes", func() {
		const validJSON = `{"key1":"value1","key2":"value2"}`
		const invalidJSON = `{{{"KEY"`

		var req *http.Request

		Context("when Content-type is not application/json", func() {
			It("returns a proper HTTPError", func() {
				req = httptest.NewRequest(http.MethodPost, "http://example.com", strings.NewReader(validJSON))
				req.Header.Add("Content-Type", "application/xml")
				_, err := util.RequestBodyToBytes(req)

				validateHTTPErrorOccured(err, http.StatusUnsupportedMediaType)
			})
		})

		Context("when reading body bytes fails", func() {
			It("returns a proper HTTPError", func() {
				req = httptest.NewRequest(http.MethodPost, "http://example.com", errorReader{})
				req.Header.Add("Content-Type", "application/json")
				_, err := util.RequestBodyToBytes(req)

				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when body is not valid JSON", func() {
			It("returns a proper HTTPError", func() {
				req = httptest.NewRequest(http.MethodPost, "http://example.com", strings.NewReader(invalidJSON))
				req.Header.Add("Content-Type", "application/json")
				_, err := util.RequestBodyToBytes(req)

				validateHTTPErrorOccured(err, http.StatusBadRequest)
			})
		})

		Context("when successful", func() {
			It("returns the []byte representation of the request body", func() {
				req = httptest.NewRequest(http.MethodPost, "http://example.com", strings.NewReader(validJSON))
				req.Header.Add("Content-Type", "application/json")
				bytes, err := util.RequestBodyToBytes(req)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(bytes)).To(Equal(validJSON))
			})
		})
	})

	Describe("BytesToObject", func() {
		const (
			testTypeValid    = `{"field1":"value1", "field2":"value2"}`
			testTypeNotValid = `{"field1":"value1"}`
			randomJSON       = `{"f1":"v1", "f2":"v2"}`
		)

		var (
			testTypeValidation   testTypeValidated
			testTypeNoValidation testTypeNotValidated
		)

		BeforeEach(func() {
			testTypeValidation = testTypeValidated{}
			testTypeNoValidation = testTypeNotValidated{}
		})

		Context("when JSON unmarshaling fails", func() {
			It("returns a proper HTTPError", func() {
				err := util.BytesToObject([]byte(randomJSON), &testTypeValidation)

				validateHTTPErrorOccured(err, http.StatusBadRequest)
			})
		})

		Context("when input validation fails", func() {
			It("returns a proper HTTPError", func() {
				err := util.BytesToObject([]byte(testTypeNotValid), &testTypeValidation)

				validateHTTPErrorOccured(err, http.StatusBadRequest)
			})
		})

		Context("when value is not InputValidator", func() {
			It("returns nil", func() {
				err := util.BytesToObject([]byte(testTypeNotValid), &testTypeNoValidation)

				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when unmarshaling and validation succeed", func() {
			It("returns nil", func() {
				err := util.BytesToObject([]byte(testTypeValid), &testTypeValidation)

				Expect(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("WriteJSON", func() {
		const testTypeValid = `{"field1":"Value1", "field2":"Value2"}`

		It("writes the code and value to the ResponseWriter and adds a Content-Type header", func() {
			expectedCode := http.StatusOK
			testValue := testTypeValidated{
				Field1: "Value1",
				Field2: "Value2",
			}
			recorder := httptest.NewRecorder()

			err := util.WriteJSON(recorder, expectedCode, testValue)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(recorder.Code).To(Equal(expectedCode))
			Expect(recorder.Body).Should(MatchJSON(testTypeValid))
			Expect(recorder.Header().Get("Content-Type")).To(Equal("application/json"))

		})
	})

	Describe("NewJSONResponse", func() {
		const testTypeValid = `{"field1":"Value1", "field2":"Value2"}`

		It("builds a web.Response containing the marshalled value with a Content-Type header", func() {
			expectedCode := http.StatusOK
			testValue := testTypeValidated{
				Field1: "Value1",
				Field2: "Value2",
			}
			response, err := util.NewJSONResponse(expectedCode, testValue)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(expectedCode))
			Expect(response.Body).Should(MatchJSON(testTypeValid))
			Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
		})

		It("builds a web.Response containing an empty response value when return code is 204 No Content", func() {
			testJSONResponse(http.StatusNoContent, util.EmptyResponseBody{}, []byte{})
		})

		It("builds a web.Response containing an empty response value when return code is 200 OK and EmptyResponseBody is provided", func() {
			testJSONResponse(http.StatusOK, util.EmptyResponseBody{}, []byte{})
		})

		It("builds a web.Response containing a non-empty response value when instead of EmptyResponseBody an empty data structure is provided", func() {
			response, err := util.NewJSONResponse(http.StatusOK, struct{}{})

			Expect(err).ShouldNot(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))
			Expect(response.Body).ShouldNot(BeEmpty())
			Expect(response.Body).ShouldNot(Equal(util.EmptyResponseBody{}))
			Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
		})
	})
})

func testJSONResponse(expectedCode int, providedBody, expectedMarshalledBody interface{}) {
	response, err := util.NewJSONResponse(expectedCode, providedBody)

	Expect(err).ShouldNot(HaveOccurred())
	Expect(response.StatusCode).To(Equal(expectedCode))
	Expect(response.Body).Should(Equal(expectedMarshalledBody))
	Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
}

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

type testTypeNotValidated struct {
	Field1 string `json:"field1"`
	Field2 string `json:"field2"`
}

type errorReader struct {
}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("error")
}
