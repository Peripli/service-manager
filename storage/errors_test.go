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

package storage

import (
	"errors"
	"testing"

	"github.com/Peripli/service-manager/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCommon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "api/Common Suite")
}

var _ = Describe("api/common", func() {

	Describe("CheckErrors", func() {
		Context("with no errors", func() {
			It("returns nil", func() {
				Expect(CheckErrors()).To(BeNil())
			})
		})
		Context("with errors", func() {
			It("should return first error", func() {
				err1 := errors.New("1")
				err2 := errors.New("oidc_authn")
				Expect(CheckErrors(err1, err2).Error()).To(Equal("1"))
			})
		})
		Context("with ResponseErrors", func() {
			It("should return the first ResponseError", func() {
				err1 := errors.New("0")
				err2 := &util.HTTPError{
					ErrorType:   "ERR",
					Description: "1",
					StatusCode:  200,
				}
				err3 := &util.HTTPError{
					ErrorType:   "ERR",
					Description: "2",
					StatusCode:  500,
				}
				Expect(CheckErrors(err1, err2, err3).Error()).To(Equal("1"))
			})
		})
	})
})
