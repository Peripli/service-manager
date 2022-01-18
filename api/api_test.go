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

package api_test

import (
	"context"
	"testing"

	"github.com/Peripli/service-manager/operations"

	"github.com/Peripli/service-manager/pkg/env/envfakes"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("API", Ordered, func() {
	var (
		mockedStorage   *storage.InterceptableTransactionalRepository
		server          *common.OAuthServer
		fakeEnvironment *envfakes.FakeEnvironment
	)

	BeforeAll(func() {
		server = common.NewOAuthServer()
	})

	AfterAll(func() {
		server.Close()
	})

	BeforeEach(func() {
		mockedStorage = storage.NewInterceptableTransactionalRepository(&storagefakes.FakeStorage{})
		fakeEnvironment = &envfakes.FakeEnvironment{}
	})

	Describe("New", func() {
		It("returns no error if creation is successful", func() {
			_, err := api.New(context.TODO(), fakeEnvironment, &api.Options{
				Repository:        mockedStorage,
				OperationSettings: &operations.Settings{},
				APISettings: &api.Settings{
					TokenIssuerURL: server.BaseURL,
					ClientID:       "sm",
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
