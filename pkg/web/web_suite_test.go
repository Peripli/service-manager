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

package web_test

import (
	"context"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestWeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Web Suite")
}

var _ = Describe("Context keys", func() {

	Context("when valid user is in context", func() {
		It("returns user and ok", func() {
			ctx := web.ContextWithUser(context.TODO(), &web.UserContext{})
			userContext, ok := web.UserFromContext(ctx)
			Expect(userContext).ToNot(BeNil())
			Expect(ok).To(BeTrue())
		})
	})

	Context("when nil user is in context", func() {
		It("returns nil and not ok", func() {
			ctx := web.ContextWithUser(context.TODO(), nil)
			userContext, ok := web.UserFromContext(ctx)
			Expect(userContext).To(BeNil())
			Expect(ok).To(BeFalse())
		})
	})

	Context("when user is not in context", func() {
		It("returns nil and not ok", func() {
			userContext, ok := web.UserFromContext(context.TODO())
			Expect(userContext).To(BeNil())
			Expect(ok).To(BeFalse())
		})
	})
})
