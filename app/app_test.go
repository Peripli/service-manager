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

package app_test

import (
	"context"
	"testing"

	"github.com/Peripli/service-manager/app"
	"github.com/Peripli/service-manager/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Suite")
}

var _ = Describe("SM test", func() {
	Context("New server", func() {
		It("should fail on config validation", func() {
			params := &app.Parameters{}
			settings := config.DefaultSettings()
			settings.Server.Port = 0
			params.Settings = settings
			_, err := app.New(context.Background(), params)
			Expect(err.Error()).To(ContainSubstring("configuration validation failed"))
		})
	})
})
