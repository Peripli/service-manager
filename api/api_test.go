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

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("API", func() {
	var (
		mockedStorage *storagefakes.FakeStorage
		server        *common.OAuthServer
	)

	BeforeSuite(func() {
		server = common.NewOAuthServer()
	})

	AfterSuite(func() {
		server.Close()
	})

	BeforeEach(func() {
		mockedStorage = &storagefakes.FakeStorage{}
	})

	Describe("New", func() {

		It("returns no error if creation is successful", func() {
			_, err := api.New(context.TODO(), mockedStorage, &api.Settings{
				TokenIssuerURL: server.BaseURL,
				ClientID:       "sm",
			}, nil)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("returns an error if creation fails", func() {
			_, err := api.New(context.TODO(), mockedStorage, &api.Settings{
				TokenIssuerURL: "http://invalidurl.com",
				ClientID:       "invalidclient",
			}, nil)
			Expect(err).Should(HaveOccurred())
		})
	})
})
